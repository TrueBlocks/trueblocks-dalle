package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	dalle "github.com/TrueBlocks/trueblocks-dalle/v6"
)

type testRecord struct {
	series string
	id     string
	label  string
}

var testRecords = []testRecord{
	{"five-tone-postal-protozoa", "02b62e935f63828bfcba3a32053df9306f30f268d15d8656618a3d6b50bcd5d4", "dead dinoflagellata biologist"},
	{"empty", "e18ce5143adc3581502cbb7c70988de13c61c16948ba57f83cf0ddc07f6fd5e5", "unconscious coryphodon gemologist"},
	{"five-tone-postal-protozoa", "8cabfc4a4c1c317db14a32e2e5036a9fb4d33486b509c90e3825f8425725b802", "yellow dinoflagellata surgeon"},
	{"five-tone-postal-protozoa", "b193c3dc843783891d51e1d8501a446b42961d5d5e82f41835ca8032e32d5fb3", "sensible phytomyxea chimney sweep"},
	{"five-tone-postal-protozoa", "794bda6b8ae529bf696ea03e4aa78525b06cd411544ca660d529835d15aa9cbb", "tenacious phytomyxea translator"},
}

var attributeNames = []string{
	"adverb",
	"adjective",
	"noun",
	"emotion",
	"occupation",
	"action",
	"artStyle1",
	"artStyle2",
	"litStyle",
	"color1",
	"color2",
	"color3",
	"viewpoint",
	"gaze",
	"composition",
	"place",
	"trope",
}

var rubricCriteria = []string{
	"Subject (creature + occupation props)",
	"Emotion",
	"Style (art style + colors)",
	"Setting (place/trope)",
	"Technical (viewpoint/composition/gaze)",
}

type evalMode string

const (
	modeArchive evalMode = "archive"
	modePrompt  evalMode = "prompt"
	modeFull    evalMode = "full"
)

type config struct {
	source  string
	output  string
	evalDir string
	dataDir string
	mode    evalMode
	stdout  io.Writer
	stderr  io.Writer
}

type metadata struct {
	Input           string            `json:"input"`
	Seed            string            `json:"seed"`
	Series          metadataSeries    `json:"series"`
	Recipe          metadataRecipe    `json:"recipe"`
	SelectedRecords []selectedRecord  `json:"selectedRecords"`
	Prompts         prompts           `json:"prompts"`
}

type metadataSeries struct {
	Name string `json:"name"`
}

type metadataRecipe struct {
	Name string `json:"name"`
}

type selectedRecord struct {
	Attribute string `json:"attribute"`
	Record    string `json:"record"`
}

type prompts struct {
	Prompt          string `json:"prompt"`
	EnhancedPrompt  string `json:"enhancedPrompt"`
}

func main() {
	os.Exit(run(os.Args[1:], config{
		source:  filepath.Join(os.Getenv("HOME"), ".local/share/trueblocks/dalle/archives"),
		output:  "scoring_sheet",
		evalDir: "dalle/eval",
		dataDir: filepath.Join(os.Getenv("HOME"), ".local/share/trueblocks/dalle"),
		mode:    modePrompt,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}))
}

func run(args []string, cfg config) int {
	if cfg.stdout == nil {
		cfg.stdout = os.Stdout
	}
	if cfg.stderr == nil {
		cfg.stderr = os.Stderr
	}

	fs := flag.NewFlagSet("evaluate-harness", flag.ContinueOnError)
	fs.SetOutput(cfg.stderr)
	source := fs.String("source", cfg.source, "source directory containing series/metadata and series/annotated")
	output := fs.String("output", cfg.output, "base name for output files")
	evalDir := fs.String("eval-dir", cfg.evalDir, "directory for evaluation output")
	dataDir := fs.String("data-dir", cfg.dataDir, "data directory for engine (parent of source archives)")
	mode := fs.String("mode", string(cfg.mode), "evaluation mode: archive, prompt (regenerate metadata), or full (regenerate metadata + image)")
	record := fs.String("record", "", "evaluate a single record as series/id/label (overrides default corpus)")
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(cfg.stderr, err)
		return 2
	}

	cfg.source = *source
	cfg.output = *output
	cfg.evalDir = *evalDir
	cfg.dataDir = *dataDir
	cfg.mode = evalMode(*mode)
	if cfg.mode != modeArchive && cfg.mode != modePrompt && cfg.mode != modeFull {
		_, _ = fmt.Fprintln(cfg.stderr, "--mode must be archive, prompt, or full")
		return 2
	}

	var records []testRecord
	if *record != "" {
		parts := strings.Split(*record, "/")
		if len(parts) != 3 {
			_, _ = fmt.Fprintln(cfg.stderr, "--record must be series/id/label")
			return 2
		}
		records = []testRecord{{parts[0], parts[1], parts[2]}}
	}

	if err := evaluate(cfg, records); err != nil {
		_, _ = fmt.Fprintln(cfg.stderr, err)
		return 1
	}
	return 0
}

func evaluate(cfg config, records []testRecord) error {
	if len(records) == 0 {
		records = testRecords
	}
	sourceDir, err := filepath.Abs(cfg.source)
	if err != nil {
		return err
	}
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source not found: %s", sourceDir)
	}

	outDir, err := filepath.Abs(cfg.evalDir)
	if err != nil {
		return err
	}
	imagesDir := filepath.Join(outDir, cfg.output+"_images")
	outPath := filepath.Join(outDir, cfg.output+"_scoring_sheet.md")

	_, _ = fmt.Fprintf(cfg.stderr, "Dalle evaluation harness\n")
	_, _ = fmt.Fprintf(cfg.stderr, "Mode: %s\n", cfg.mode)
	_, _ = fmt.Fprintf(cfg.stderr, "Source: %s\n", sourceDir)
	_, _ = fmt.Fprintf(cfg.stderr, "Output: %s\n\n", outPath)

	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return err
	}

	var engine *dalle.Engine
	if cfg.mode == modePrompt || cfg.mode == modeFull {
		engine, err = newEngine(cfg)
		if err != nil {
			return err
		}
	}

	results := make([]recordResult, 0, len(records))
	for i, rec := range records {
		_, _ = fmt.Fprintf(cfg.stderr, "Loading record %d/%d: %s\n", i+1, len(records), rec.label)

		if cfg.mode == modePrompt || cfg.mode == modeFull {
			_, _ = fmt.Fprintf(cfg.stderr, "  Regenerating %s...\n", func() string {
				if cfg.mode == modeFull {
					return "metadata and image"
				}
				return "metadata"
			}())
			if err := regenerateRecord(engine, sourceDir, rec, cfg.mode == modeFull); err != nil {
				return err
			}
		}

		meta, err := loadMetadata(sourceDir, rec.series, rec.id)
		if err != nil {
			return err
		}

		selected := selectedToMap(meta.SelectedRecords)
		prompts := meta.Prompts

		_, _ = fmt.Fprintf(cfg.stderr, "  Checking %d attributes...\n", len(attributeNames))
		checks := checkAttributes(selected, prompts)

		_, _ = fmt.Fprintf(cfg.stderr, "  Copying annotated image...\n")
		imageRel, err := copyImage(sourceDir, rec.series, rec.id, imagesDir, outDir)
		if err != nil {
			return err
		}

		results = append(results, recordResult{
			series:        rec.series,
			id:            rec.id,
			label:         rec.label,
			input:         meta.Input,
			selected:      selected,
			prompts:       prompts,
			checks:        checks,
			imageRelative: imageRel,
		})
	}

	_, _ = fmt.Fprintf(cfg.stderr, "\nWriting scoring sheet...\n")
	if err := writeScoringSheet(results, sourceDir, outPath); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cfg.stderr, "Wrote %s\n", outPath)

	return nil
}

type recordResult struct {
	series        string
	id            string
	label         string
	input         string
	selected      map[string]string
	prompts       prompts
	checks        map[string]bool
	imageRelative string
}

func newEngine(cfg config) (*dalle.Engine, error) {
	dataDir := cfg.dataDir
	if dataDir == "" {
		dataDir = filepath.Dir(cfg.source)
	}
	return dalle.New(dalle.Config{DataDir: dataDir})
}

func regenerateRecord(engine *dalle.Engine, sourceDir string, rec testRecord, withImage bool) error {
	meta, err := loadMetadata(sourceDir, rec.series, rec.id)
	if err != nil {
		return err
	}

	backstyle := ""
	for _, sr := range meta.SelectedRecords {
		if sr.Attribute == "backStyle" {
			backstyle = sr.Record
			break
		}
	}

	result, err := engine.Generate(dalle.GenerateRequest{
		Input:     meta.Input,
		Seed:      meta.Seed,
		Series:    meta.Series.Name,
		Recipe:    meta.Recipe.Name,
		Backstyle: backstyle,
		Enhance:   strings.TrimSpace(meta.Prompts.EnhancedPrompt) != "",
		Image:     withImage,
		Annotate:  withImage,
		Force:     true,
	})
	if err != nil {
		return err
	}

	metadataDst := filepath.Join(sourceDir, rec.series, "metadata", rec.id+".json")
	if err := os.MkdirAll(filepath.Dir(metadataDst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(result.MetadataPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataDst, data, 0o644); err != nil {
		return err
	}

	if withImage && result.AnnotatedPath != "" {
		annotatedDst := filepath.Join(sourceDir, rec.series, "annotated", rec.id+".png")
		if err := os.MkdirAll(filepath.Dir(annotatedDst), 0o755); err != nil {
			return err
		}
		imgData, err := os.ReadFile(result.AnnotatedPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(annotatedDst, imgData, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func loadMetadata(sourceDir, series, id string) (metadata, error) {
	path := filepath.Join(sourceDir, series, "metadata", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return metadata{}, err
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return metadata{}, err
	}
	return meta, nil
}

func selectedToMap(selected []selectedRecord) map[string]string {
	result := make(map[string]string, len(selected))
	for _, s := range selected {
		result[s.Attribute] = s.Record
	}
	return result
}

func checkAttributes(selected map[string]string, prompts prompts) map[string]bool {
	checks := make(map[string]bool, len(attributeNames))
	haystack := strings.ToLower(prompts.Prompt + " " + prompts.EnhancedPrompt)
	for _, attr := range attributeNames {
		checks[attr] = attributePresent(attr, selected, haystack)
	}
	return checks
}

func attributePresent(attr string, selected map[string]string, haystack string) bool {
	record, ok := selected[attr]
	if !ok {
		return false
	}
	if strings.TrimSpace(strings.ToLower(record)) == "none" {
		return true
	}
	if strings.TrimSpace(record) == "" {
		return true
	}

	if strings.HasPrefix(attr, "color") {
		for part := range strings.SplitSeq(record, ",") {
			p := strings.TrimSpace(strings.ToLower(part))
			if p != "" && strings.Contains(haystack, p) {
				return true
			}
		}
		return false
	}

	term := extractMainTerm(record)
	return strings.Contains(haystack, term)
}

func extractMainTerm(record string) string {
	for part := range strings.SplitSeq(record, ",") {
		return strings.TrimSpace(strings.ToLower(part))
	}
	return ""
}

func copyImage(sourceDir, series, id, imagesDir, outDir string) (string, error) {
	src := filepath.Join(sourceDir, series, "annotated", id+".png")
	dst := filepath.Join(imagesDir, series+"_"+id+".png")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return "", nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(outDir, dst)
	if err != nil {
		return "", err
	}
	return rel, nil
}

func writeScoringSheet(results []recordResult, sourceLabel, outPath string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Dalle Prompt-Image Alignment Scoring Sheet\n\n")
	fmt.Fprintf(&b, "**Generated:** from `%s`.\n", sourceLabel)
	fmt.Fprintf(&b, "**Instructions:** For each image, score 1–5 on each rubric item. Total = 5–25.\n\n")
	fmt.Fprintf(&b, "---\n\n")

	for i, r := range results {
		if i > 0 {
			fmt.Fprintf(&b, "---\n\n")
		}
		fmt.Fprintf(&b, "## %s\n", r.label)
		fmt.Fprintf(&b, "**Series:** `%s`  \n", r.series)
		fmt.Fprintf(&b, "**ID:** `%s`  \n", r.id)
		fmt.Fprintf(&b, "**Input:** *%s*\n\n", r.input)
		fmt.Fprintf(&b, "![%s](%s)\n\n", r.label, r.imageRelative)

		fmt.Fprintf(&b, "### Prompt-Level Attribute Checklist\n\n")
		fmt.Fprintf(&b, "| Attribute | Present | Value |\n")
		fmt.Fprintf(&b, "|---|---|---|\n")
		present := 0
		for _, attr := range attributeNames {
			check := r.checks[attr]
			value := "—"
			if v, ok := r.selected[attr]; ok {
				value = v
			}
			shortValue := value
			if value != "—" {
				for part := range strings.SplitSeq(value, ",") {
					shortValue = part
					break
				}
			}
			mark := "❌"
			if check {
				mark = "✅"
				present++
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", attr, mark, shortValue)
		}
		fmt.Fprintf(&b, "| **Total** | **%d/%d** | |\n\n", present, len(attributeNames))

		fmt.Fprintf(&b, "### Prompts\n\n")
		fmt.Fprintf(&b, "**Basic:**\n")
		fmt.Fprintf(&b, "```\n%s\n```\n\n", r.prompts.Prompt)
		fmt.Fprintf(&b, "**Enhanced:**\n")
		enhanced := r.prompts.EnhancedPrompt
		if strings.TrimSpace(enhanced) == "" {
			enhanced = "N/A"
		}
		fmt.Fprintf(&b, "```\n%s\n```\n\n", enhanced)

		fmt.Fprintf(&b, "### Image-Level Rubric\n\n")
		fmt.Fprintf(&b, "| Criterion | Score (1–5) | Notes |\n")
		fmt.Fprintf(&b, "|---|---|---|\n")
		for _, criterion := range rubricCriteria {
			fmt.Fprintf(&b, "| %s | | |\n", criterion)
		}
		fmt.Fprintf(&b, "| **Total** | **/25** | |\n\n")
	}
	fmt.Fprintf(&b, "---\n")

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}
