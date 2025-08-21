package dalle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Phase represents a canonical generation phase.
type Phase string

const (
	PhaseSetup         Phase = "setup"
	PhaseBasePrompts   Phase = "base_prompts"
	PhaseEnhance       Phase = "enhance_prompt"
	PhaseImagePrep     Phase = "image_prep"
	PhaseImageWait     Phase = "image_wait"
	PhaseImageDownload Phase = "image_download"
	PhaseAnnotate      Phase = "annotate"
	PhaseCompleted     Phase = "completed"
)

// OrderedPhases defines the progression order (including completed terminal for simplicity).
var OrderedPhases = []Phase{
	PhaseSetup, PhaseBasePrompts, PhaseEnhance, PhaseImagePrep, PhaseImageWait, PhaseImageDownload, PhaseAnnotate, PhaseCompleted,
}

// PhaseTiming captures timing and status per phase.
type PhaseTiming struct {
	Name      Phase  `json:"name"`
	StartedNs int64  `json:"startedNs"`
	EndedNs   int64  `json:"endedNs"`
	Skipped   bool   `json:"skipped"`
	Error     string `json:"error"`
}

// ProgressReport is a snapshot of a run.
type ProgressReport struct {
	Series        string                  `json:"series"`
	Address       string                  `json:"address"`
	Current       Phase                   `json:"currentPhase"`
	StartedNs     int64                   `json:"startedNs"`
	Percent       float64                 `json:"percent"`
	ETASeconds    float64                 `json:"etaSeconds"`
	Done          bool                    `json:"done"`
	Error         string                  `json:"error"`
	CacheHit      bool                    `json:"cacheHit"`
	Phases        []*PhaseTiming          `json:"phases"`
	DalleDress    *DalleDress             `json:"dalleDress"`
	PhaseAverages map[Phase]time.Duration `json:"phaseAverages"`
}

// timeSource allows test control of time.
type timeSource interface{ Now() time.Time }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// emaAlpha is fixed per design.
const emaAlpha = 0.2

// phaseAverage holds running average in nanoseconds and count.
type phaseAverage struct {
	Count int64 `json:"count"`
	AvgNs int64 `json:"avgNs"`
}

// metricsPersistence stores global metrics.
type metricsPersistence struct {
	Version        string                  `json:"version"`
	Phase          map[Phase]*phaseAverage `json:"phaseAverages"`
	GenerationRuns int64                   `json:"generationRuns"`
	CacheHits      int64                   `json:"cacheHits"`
}

// progressRun is internal mutable state for one run.
type progressRun struct {
	series   string
	address  string
	phases   map[Phase]*PhaseTiming
	order    []Phase
	dress    *DalleDress
	start    time.Time
	current  Phase
	done     bool
	cacheHit bool
	err      string
}

// ProgressManager manages concurrent runs and global averages.
type ProgressManager struct {
	mu      sync.Mutex
	runs    map[string]*progressRun // key series:address
	metrics metricsPersistence
	clock   timeSource
}

// global singleton (can be replaced in tests)
var progressMgr = &ProgressManager{runs: map[string]*progressRun{}, metrics: metricsPersistence{Version: "v1", Phase: map[Phase]*phaseAverage{}}, clock: realClock{}}

const metricsDirName = "metrics"
const metricsFile = "progress_phase_stats.json"
const archiveEnv = "DALLESERVER_ARCHIVE_RUNS"

var metricsDir = metricsDirName // overridden at runtime to DataDir/metrics via SetMetricsDir

// SetMetricsBaseDir sets the root directory where the metrics folder ("metrics") lives.
// Call early during application startup (e.g., after resolving DataDir). Tests may also
// override to temp directories for isolation.
func SetMetricsDir(base string) {
	if base == "" {
		metricsDir = metricsDirName
		return
	}
	metricsDir = filepath.Join(base, metricsDirName)
}

var metricsLoaded bool

func loadMetricsLocked(pm *ProgressManager) {
	if metricsLoaded {
		return
	}
	path := filepath.Join(metricsDir, metricsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		metricsLoaded = true
		return
	}
	var m metricsPersistence
	if json.Unmarshal(data, &m) == nil && m.Version == "v1" {
		if m.Phase == nil {
			m.Phase = map[Phase]*phaseAverage{}
		}
		pm.metrics = m
	}
	metricsLoaded = true
}

func saveMetricsLocked(pm *ProgressManager) {
	if pm.metrics.Version == "" {
		pm.metrics.Version = "v1"
	}
	_ = os.MkdirAll(metricsDir, 0o755)
	path := filepath.Join(metricsDir, metricsFile)
	b, err := json.MarshalIndent(pm.metrics, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// key builds a composite key.
func key(series, addr string) string { return series + ":" + addr }

// StartRun initializes a run if not present.
func (pm *ProgressManager) StartRun(series, addr string, dress *DalleDress) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	k := key(series, addr)
	if _, exists := pm.runs[k]; exists {
		return
	}
	run := &progressRun{series: series, address: addr, dress: dress, phases: map[Phase]*PhaseTiming{}, order: OrderedPhases, start: pm.clock.Now(), current: PhaseSetup}
	for _, ph := range OrderedPhases {
		run.phases[ph] = &PhaseTiming{Name: ph}
	}
	run.phases[PhaseSetup].StartedNs = run.start.UnixNano()
	pm.runs[k] = run
}

// Transition ends current phase and starts the next.
func (pm *ProgressManager) Transition(series, addr string, ph Phase) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil || run.done {
		return
	}
	now := pm.clock.Now().UnixNano()
	cur := run.phases[run.current]
	if cur.StartedNs != 0 && cur.EndedNs == 0 && run.current != ph {
		cur.EndedNs = now
		pm.updateAverageLocked(run, run.current, cur)
	}
	next := run.phases[ph]
	if next.StartedNs == 0 {
		next.StartedNs = now
	}
	run.current = ph
}

// Complete finalizes the run.
func (pm *ProgressManager) Complete(series, addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil || run.done {
		return
	}
	now := pm.clock.Now().UnixNano()
	cur := run.phases[run.current]
	if cur.StartedNs != 0 && cur.EndedNs == 0 {
		cur.EndedNs = now
		pm.updateAverageLocked(run, run.current, cur)
	}
	comp := run.phases[PhaseCompleted]
	if comp.StartedNs == 0 {
		comp.StartedNs = now
	}
	if comp.EndedNs == 0 {
		comp.EndedNs = now
	}
	run.current = PhaseCompleted
	run.done = true
	if !run.cacheHit {
		pm.metrics.GenerationRuns++
		// Persist metrics after a new full generation
		saveMetricsLocked(pm)
	} else {
		// Cache hit completion: ensure CacheHits counter is persisted even if no averages updated
		saveMetricsLocked(pm)
	}
	pm.maybeArchiveRunLocked(run)
}

// Fail marks an error completion.
func (pm *ProgressManager) Fail(series, addr string, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil || run.done {
		return
	}
	run.err = err.Error()
	run.done = true
	now := pm.clock.Now().UnixNano()
	cur := run.phases[run.current]
	if cur.StartedNs != 0 && cur.EndedNs == 0 {
		cur.EndedNs = now
	}
	comp := run.phases[PhaseCompleted]
	if comp.StartedNs == 0 {
		comp.StartedNs = now
	}
	if comp.EndedNs == 0 {
		comp.EndedNs = now
	}
	pm.maybeArchiveRunLocked(run)
}

// Skip marks a phase skipped.
func (pm *ProgressManager) Skip(series, addr string, ph Phase) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil {
		return
	}
	p := run.phases[ph]
	if p.Skipped {
		return
	}
	p.Skipped = true
	if p.StartedNs == 0 {
		p.StartedNs = pm.clock.Now().UnixNano()
	}
	if p.EndedNs == 0 {
		p.EndedNs = p.StartedNs
	}
}

// MarkCacheHit notes a cache hit.
func (pm *ProgressManager) MarkCacheHit(series, addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil {
		return
	}
	if !run.cacheHit {
		run.cacheHit = true
		pm.metrics.CacheHits++
		// Persist immediately so pure cache-hit runs (with no phase averages) still write metrics file
		saveMetricsLocked(pm)
	}
}

// GetReport returns a snapshot; discards run if done.
func (pm *ProgressManager) GetReport(series, addr string) *ProgressReport {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	run := pm.runs[key(series, addr)]
	if run == nil {
		return nil
	}
	pr := &ProgressReport{Series: run.series, Address: run.address, Current: run.current, StartedNs: run.start.UnixNano(), Done: run.done, Error: run.err, CacheHit: run.cacheHit}
	for _, ph := range run.order {
		p := run.phases[ph]
		cp := *p
		pr.Phases = append(pr.Phases, &cp)
	}
	// Reuse the live DalleDress pointer to avoid duplication; callers should treat as read-only.
	pr.DalleDress = run.dress
	pr.PhaseAverages = map[Phase]time.Duration{}
	for ph, avg := range pm.metrics.Phase {
		if avg.Count > 0 {
			pr.PhaseAverages[ph] = time.Duration(avg.AvgNs)
		}
	}
	pm.computePercentETA(pr, run)
	if run.done {
		delete(pm.runs, key(series, addr))
	}
	return pr
}

// maybeArchiveRunLocked writes a JSON snapshot of a completed run if archiving is enabled.
func (pm *ProgressManager) maybeArchiveRunLocked(run *progressRun) {
	if os.Getenv(archiveEnv) != "1" || !run.done {
		return
	}
	// Build a snapshot similar to GetReport without deleting the run (caller will manage lifecycle)
	pr := &ProgressReport{Series: run.series, Address: run.address, Current: run.current, StartedNs: run.start.UnixNano(), Done: run.done, Error: run.err, CacheHit: run.cacheHit}
	for _, ph := range run.order {
		p := run.phases[ph]
		cp := *p
		pr.Phases = append(pr.Phases, &cp)
	}
	pr.DalleDress = run.dress
	pr.PhaseAverages = map[Phase]time.Duration{}
	for ph, avg := range pm.metrics.Phase {
		if avg.Count > 0 {
			pr.PhaseAverages[ph] = time.Duration(avg.AvgNs)
		}
	}
	pm.computePercentETA(pr, run)
	_ = os.MkdirAll(filepath.Join(metricsDir, "runs"), 0o755)
	fn := fmt.Sprintf("%s_%s_%d.json", run.series, run.address, time.Now().Unix())
	path := filepath.Join(metricsDir, "runs", fn)
	if b, err := json.MarshalIndent(pr, "", "  "); err == nil {
		_ = os.WriteFile(path, b, 0o644)
	}
}

// UpdateDress mutates the underlying DalleDress for a run under the manager lock.
// Use for progress-relevant field changes to reduce race potential when reusing the pointer.
func (pm *ProgressManager) UpdateDress(series, addr string, fn func(*DalleDress)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if run := pm.runs[key(series, addr)]; run != nil && run.dress != nil {
		fn(run.dress)
	}
}

// computePercentETA fills percent and ETA.
func (pm *ProgressManager) computePercentETA(pr *ProgressReport, run *progressRun) {
	var total, doneDur, currentElapsed time.Duration
	now := pm.clock.Now()
	for _, ph := range run.order {
		if ph == PhaseCompleted {
			continue
		}
		if a := pm.metrics.Phase[ph]; a != nil && a.Count > 0 {
			total += time.Duration(a.AvgNs)
		}
	}
	if total == 0 {
		pr.Percent = 0
		pr.ETASeconds = 0
		return
	}
	for _, ph := range run.order {
		if ph == PhaseCompleted {
			continue
		}
		pt := run.phases[ph]
		avg := time.Duration(0)
		if a := pm.metrics.Phase[ph]; a != nil && a.Count > 0 {
			avg = time.Duration(a.AvgNs)
		}
		if ph == run.current {
			if pt.StartedNs > 0 {
				currentElapsed = now.Sub(time.Unix(0, pt.StartedNs))
			}
			if avg > 0 && currentElapsed > avg {
				currentElapsed = avg
			}
			doneDur += currentElapsed
			break
		} else {
			if pt.EndedNs > 0 && avg > 0 {
				doneDur += avg
			}
		}
	}
	pr.Percent = (float64(doneDur) / float64(total)) * 100
	remaining := total - doneDur
	if remaining < 0 {
		remaining = 0
	}
	pr.ETASeconds = remaining.Seconds()
}

// updateAverageLocked updates EMA for a finished phase if eligible.
func (pm *ProgressManager) updateAverageLocked(run *progressRun, ph Phase, t *PhaseTiming) {
	if run.cacheHit || run.err != "" || t.Skipped || t.StartedNs == 0 || t.EndedNs == 0 {
		return
	}
	if !metricsLoaded {
		loadMetricsLocked(pm)
	}
	dur := t.EndedNs - t.StartedNs
	if dur <= 0 {
		return
	}
	rec := pm.metrics.Phase[ph]
	if rec == nil {
		rec = &phaseAverage{Count: 1, AvgNs: dur}
		pm.metrics.Phase[ph] = rec
		saveMetricsLocked(pm)
		return
	}
	rec.AvgNs = int64(float64(dur)*emaAlpha + float64(rec.AvgNs)*(1-emaAlpha))
	rec.Count++
	saveMetricsLocked(pm)
}

// MarshalProgressReport pretty prints a report.
func MarshalProgressReport(pr *ProgressReport) []byte {
	b, _ := json.MarshalIndent(pr, "", "  ")
	return b
}

// GetProgress returns the current progress report (public helper).
func GetProgress(series, addr string) *ProgressReport { return progressMgr.GetReport(series, addr) }

// ForceMetricsSave forces metrics write (for tests).
func ForceMetricsSave() {
	progressMgr.mu.Lock()
	defer progressMgr.mu.Unlock()
	saveMetricsLocked(progressMgr)
}

// ResetMetricsForTest clears metrics (for tests).
func ResetMetricsForTest() {
	progressMgr.mu.Lock()
	defer progressMgr.mu.Unlock()
	metricsLoaded = false
	progressMgr.metrics = metricsPersistence{Version: "v1", Phase: map[Phase]*phaseAverage{}}
}
