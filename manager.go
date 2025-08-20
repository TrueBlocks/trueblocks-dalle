package dalle

// Manager and locking utilities migrated from server integration layer so the server
// can call directly into the dalle module without an adapter facade.

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/walk"
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

// getContext returns (and possibly creates) a managed Context for a series.
func getContext(series, outputDir string) (*managedContext, error) {
	contextManager.Lock()
	defer contextManager.Unlock()
	if mc, ok := contextManager.items[series]; ok {
		mc.lastUsed = time.Now()
		bumpOrder(series)
		return mc, nil
	}
	c := NewContext(outputDir)
	seriesJSON := filepath.Join(outputDir, "series", series+".json")
	if file.FileExists(seriesJSON) {
		if ser, err := c.LoadSeries(); err == nil {
			ser.Suffix = series
			c.Series = ser
		} else {
			c.Series.Suffix = series
		}
	} else {
		c.Series.Suffix = series
	}
	mc := &managedContext{ctx: c, series: series, lastUsed: time.Now()}
	contextManager.items[series] = mc
	contextManager.order = append(contextManager.order, series)
	enforceContextLimits()
	return mc, nil
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

// GenerateAnnotatedImage builds (and optionally generates) an annotated image path.
// The image generation step is skipped if skipImage is true.
func GenerateAnnotatedImage(series, address, outputDir string, skipImage bool, lockTTL time.Duration) (string, error) {
	start := time.Now()
	logger.Info("GenerateAnnotatedImage:start", series, address)
	if address == "" {
		return "", errors.New("address required")
	}
	cleanupLocks()
	if lockTTL <= 0 {
		lockTTL = 5 * time.Minute
	}
	key := series + ":" + address
	if !acquireLock(key, lockTTL) {
		return filepath.Join(outputDir, series, "annotated", address+".png"), nil
	}
	defer releaseLock(key)
	mc, err := getContext(series, outputDir)
	if err != nil {
		return "", err
	}
	if _, err := mc.ctx.MakeDalleDress(address); err != nil {
		return "", err
	}
	if !skipImage {
		if _, err := mc.ctx.GenerateImage(address); err != nil {
			return "", err
		}
	} else {
		logger.Info("GenerateAnnotatedImage:skipImage true - not calling GenerateImage", series, address)
	}
	out := filepath.Join(outputDir, series, "annotated", address+".png")
	logger.Info("GenerateAnnotatedImage:end", series, address, "elapsed", time.Since(start).String())
	return out, nil
}

// ListSeries returns the list of existing series (json files) beneath outputDir/series.
func ListSeries(outputDir string) []string {
	list := []string{}
	base := filepath.Join(outputDir, "series")
	vFunc := func(fn string, vP any) (bool, error) {
		if strings.HasSuffix(fn, ".json") {
			fn = strings.ReplaceAll(fn, base+"/", "")
			fn = strings.ReplaceAll(fn, ".json", "")
			list = append(list, fn)
		}
		return true, nil
	}
	_ = walk.ForEveryFileInFolder(base, vFunc, nil)
	return list
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
