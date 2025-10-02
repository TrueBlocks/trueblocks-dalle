package dalle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/walk"
	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/progress"
	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/storage"
)

// managedContext wraps a dalle Context with bookkeeping for LRU + TTL.
type managedContext struct {
	ctx      *Context
	series   string
	lastUsed time.Time
}

var contextManager = struct {
	sync.Mutex
	items map[string]*managedContext
	order []string
}{items: map[string]*managedContext{}, order: []string{}}

// ManagerOptions controls cache sizing and expiration.
type ManagerOptions struct {
	MaxContexts int
	ContextTTL  time.Duration
}

var managerOptions = ManagerOptions{MaxContexts: 20, ContextTTL: 30 * time.Minute}

// ConfigureManager allows callers to override default manager options.
func ConfigureManager(opts ManagerOptions) {
	if opts.MaxContexts > 0 {
		managerOptions.MaxContexts = opts.MaxContexts
	}
	if opts.ContextTTL > 0 {
		managerOptions.ContextTTL = opts.ContextTTL
	}
}

// requestLocks guards concurrent image generations per (series,address).
var requestLocks = struct {
	sync.Mutex
	items map[string]time.Time
}{items: map[string]time.Time{}}

// acquireLock obtains a lock with TTL; returns false if held and not expired.
func acquireLock(key string, ttl time.Duration) bool {
	now := time.Now()
	requestLocks.Lock()
	defer requestLocks.Unlock()
	if exp, ok := requestLocks.items[key]; ok && now.Before(exp) {
		return false
	}
	requestLocks.items[key] = now.Add(ttl)
	return true
}

func releaseLock(key string) {
	requestLocks.Lock()
	delete(requestLocks.items, key)
	requestLocks.Unlock()
}

func cleanupLocks() {
	now := time.Now()
	requestLocks.Lock()
	for k, exp := range requestLocks.items {
		if now.After(exp) {
			delete(requestLocks.items, k)
		}
	}
	requestLocks.Unlock()
}

func bumpOrder(series string) {
	for i, s := range contextManager.order {
		if s == series {
			contextManager.order = append(append(contextManager.order[:i], contextManager.order[i+1:]...), series)
			return
		}
	}
}

func enforceContextLimits() {
	maxContexts := managerOptions.MaxContexts
	ttl := managerOptions.ContextTTL
	now := time.Now()
	changed := false
	for k, v := range contextManager.items {
		if now.Sub(v.lastUsed) > ttl {
			delete(contextManager.items, k)
			changed = true
		}
	}
	if changed {
		rebuildOrder()
	}
	if len(contextManager.items) <= maxContexts {
		return
	}
	pairs := make([]struct {
		k string
		t time.Time
	}, 0, len(contextManager.items))
	for k, v := range contextManager.items {
		pairs = append(pairs, struct {
			k string
			t time.Time
		}{k, v.lastUsed})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].t.Before(pairs[j].t) })
	overflow := len(contextManager.items) - maxContexts
	for i := 0; i < overflow; i++ {
		delete(contextManager.items, pairs[i].k)
	}
	rebuildOrder()
}

func rebuildOrder() {
	contextManager.order = contextManager.order[:0]
	for k := range contextManager.items {
		contextManager.order = append(contextManager.order, k)
	}
}

// getContext returns (and possibly creates) a managed Context for a series.
func getContext(series string) (*managedContext, error) {
	contextManager.Lock()
	defer contextManager.Unlock()
	logger.Info(fmt.Sprintf("getContext: Requested context for series '%s'", series))

	if mc, ok := contextManager.items[series]; ok {
		logger.Info(fmt.Sprintf("getContext: Found existing context for series '%s'", series))
		mc.lastUsed = time.Now()
		bumpOrder(series)
		return mc, nil
	}

	logger.Info(fmt.Sprintf("getContext: Creating new context for series '%s'", series))
	c := NewContext()
	if s, err := c.loadSeries(series); err == nil {
		logger.Info(fmt.Sprintf("getContext: Successfully loaded series data for '%s'", series))
		c.Series = s
		if err := c.ReloadDatabases(series); err != nil {
			logger.Error(fmt.Sprintf("getContext: Failed to reload databases with series '%s': %v", series, err))
		} else {
			logger.Info(fmt.Sprintf("getContext: Successfully reloaded databases with series '%s' filters", series))
		}
	} else {
		logger.Error(fmt.Sprintf("getContext: Failed to load series data for '%s': %v", series, err))
	}
	c.Series.Suffix = series

	mc := &managedContext{ctx: c, series: series, lastUsed: time.Now()}
	contextManager.items[series] = mc
	contextManager.order = append(contextManager.order, series)
	enforceContextLimits()
	logger.Info(fmt.Sprintf("getContext: Context created and cached for series '%s'", series))
	return mc, nil
}

// GenerateAnnotatedImage builds (and optionally generates) an annotated image path.
// The image generation step is skipped if skipImage is true.
func GenerateAnnotatedImage(series, address string, skipImage bool, lockTTL time.Duration) (string, error) {
	return GenerateAnnotatedImageWithBaseURL(series, address, skipImage, lockTTL, "")
}

// GenerateAnnotatedImageWithBaseURL builds (and optionally generates) an annotated image path with a specific base URL.
func GenerateAnnotatedImageWithBaseURL(series, address string, skipImage bool, lockTTL time.Duration, baseURL string) (string, error) {
	start := time.Now()
	logger.Info("annotated.build.start", "series", series, "addr", address, "skipImage", skipImage)
	logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Starting generation for series='%s', address='%s', skipImage=%v", series, address, skipImage))

	if address == "" {
		return "", errors.New("address required")
	}
	cleanupLocks()
	if lockTTL <= 0 {
		lockTTL = 5 * time.Minute
	}
	key := series + ":" + address
	annotatedPath := filepath.Join(storage.OutputDir(), series, "annotated", address+".png")
	logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Target annotated path: %s", annotatedPath))

	// Fast path: if annotated file exists, treat as cache hit (do not acquire new run if another generation not in progress)
	if file.FileExists(annotatedPath) {
		logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Found existing annotated image at %s (cache hit)", annotatedPath))
		// Start a minimal completed progress run if none exists yet
		progressMgr := progress.GetProgressManager()
		if progressMgr.GetReport(series, address) == nil { // no active run
			// We need a DalleDress to attach; attempt to build (cached or new) context
			if mc, err2 := getContext(series); err2 == nil {
				if dd, err3 := mc.ctx.MakeDalleDress(address); err3 == nil {
					progressMgr.StartRun(series, address, dd)
					progressMgr.MarkCacheHit(series, address)
					progressMgr.Transition(series, address, progress.PhaseBasePrompts)
					progressMgr.Transition(series, address, progress.PhaseCompleted)
					dd.CacheHit = true
					dd.Completed = true
					progressMgr.Complete(series, address)
				}
			}
		}
		return annotatedPath, nil
	}
	if !acquireLock(key, lockTTL) {
		logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Could not acquire lock for key '%s', another generation may be in progress", key))
		// Existing run or completed image (lock held by another or recently): return path
		return annotatedPath, nil
	}
	logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Acquired lock for key '%s', proceeding with generation", key))

	defer releaseLock(key)
	mc, err := getContext(series)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Failed to get context for series '%s': %v", series, err))
		return "", err
	}
	logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Got context for series '%s', creating DalleDress for address '%s'", series, address))

	dd, err := mc.ctx.MakeDalleDress(address)
	if err != nil {
		logger.Error(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Failed to create DalleDress for series '%s', address '%s': %v", series, address, err))
		return "", err
	}
	logger.Info(fmt.Sprintf("GenerateAnnotatedImageWithBaseURL: Created DalleDress for address '%s' - Prompt: '%s', EnhancedPrompt: '%s', Series: '%s'", address, dd.Prompt, dd.EnhancedPrompt, dd.Series))
	// Start progress tracking (setup already implicitly started when struct created)
	progressMgr := progress.GetProgressManager()
	progressMgr.StartRun(series, address, dd)
	progressMgr.Transition(series, address, progress.PhaseBasePrompts)
	if !skipImage {
		progressMgr.Transition(series, address, progress.PhaseEnhance)
		if _, err := mc.ctx.GenerateImageWithBaseURL(address, baseURL); err != nil {
			progressMgr.Fail(series, address, err)
			return "", err
		}
		// ImagePrep transition happens inside GenerateImage via RequestImage modifications
	} else {
		logger.Info("annotated.build.skip_image", "series", series, "addr", address)
		progressMgr.Skip(series, address, progress.PhaseEnhance)
		progressMgr.Skip(series, address, progress.PhaseImagePrep)
		progressMgr.Skip(series, address, progress.PhaseImageWait)
		progressMgr.Skip(series, address, progress.PhaseImageDownload)
	}
	// Annotation phase managed by RequestImage now; if skipImage we simulate it immediately
	if skipImage {
		progressMgr.Transition(series, address, progress.PhaseAnnotate)
	}
	out := filepath.Join(storage.OutputDir(), series, "annotated", address+".png")
	// Mark completion
	dd.Completed = true
	progressMgr.Transition(series, address, progress.PhaseCompleted)
	progressMgr.Complete(series, address)
	logger.InfoG("annotated.build.end", "series", series, "addr", address, "durMs", time.Since(start).Milliseconds())
	return out, nil
}

// ListSeries returns the list of existing series (json files) beneath output Dir/series.
func ListSeries() []string {
	seriesDir := storage.SeriesDir()
	logger.Info(fmt.Sprintf("ListSeries: Scanning directory: %s", seriesDir))

	list := []string{}
	vFunc := func(fn string, vP any) (bool, error) {
		logger.Info(fmt.Sprintf("ListSeries: Found file: %s", fn))
		if strings.HasSuffix(fn, ".json") {
			original := fn
			fn = strings.ReplaceAll(fn, storage.SeriesDir()+"/", "")
			fn = strings.ReplaceAll(fn, ".json", "")
			logger.Info(fmt.Sprintf("ListSeries: Adding series '%s' (from file: %s)", fn, original))
			list = append(list, fn)
		} else {
			logger.Info(fmt.Sprintf("ListSeries: Skipping non-JSON file: %s", fn))
		}
		return true, nil
	}
	err := walk.ForEveryFileInFolder(storage.SeriesDir(), vFunc, nil)
	if err != nil {
		logger.Warn(fmt.Sprintf("ListSeries: Error walking directory %s: %v", seriesDir, err))
	}

	logger.Info(fmt.Sprintf("ListSeries: Completed scan, found %d series: %v", len(list), list))
	return list
}

// ResetContextManagerForTest clears context cache (test helper).
func ResetContextManagerForTest() {
	contextManager.Lock()
	contextManager.items = map[string]*managedContext{}
	contextManager.order = []string{}
	contextManager.Unlock()
}

// IsValidSeries determines whether a requested series is valid given an optional list.
func IsValidSeries(series string, list []string) bool {
	if len(list) == 0 {
		return true
	}
	for _, s := range list {
		if s == series {
			return true
		}
	}
	return false
}

// ContextCount (testing) returns number of cached contexts.
func ContextCount() int {
	contextManager.Lock()
	defer contextManager.Unlock()
	return len(contextManager.items)
}

// Clean removes generated images and data for a given series and address.
func Clean(series, address string) {
	outputDir := storage.OutputDir()
	baseDir := filepath.Join(outputDir, series)

	// These are the various paths we wish to remove
	paths := []string{
		filepath.Join(baseDir, "annotated", address+".png"),
		filepath.Join(baseDir, "selector", address+".json"),
		filepath.Join(baseDir, "generated", address+".png"),
		filepath.Join(baseDir, "audio", address+".mp3"),
	}

	for _, dir := range []string{"data", "title", "terse", "prompt", "enhanced"} {
		paths = append(paths, filepath.Join(baseDir, dir, address+".txt"))
	}

	for _, p := range paths {
		_ = os.Remove(p)
	}
}
