package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	dalle "github.com/TrueBlocks/trueblocks-dalle/v6"
)

type cliConfig struct {
	dataDir         string
	providerBaseURL string
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
}

func main() {
	os.Exit(run(os.Args[1:], cliConfig{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}))
}

func run(args []string, config cliConfig) int {
	if config.stdin == nil {
		config.stdin = os.Stdin
	}
	if config.stdout == nil {
		config.stdout = os.Stdout
	}
	if config.stderr == nil {
		config.stderr = os.Stderr
	}
	global, remaining, err := parseGlobalFlags(args)
	if err != nil {
		writeError(config.stderr, err)
		return 2
	}
	config.dataDir = global.dataDir
	config.providerBaseURL = global.providerBaseURL
	if len(remaining) == 0 {
		writeError(config.stderr, fmt.Errorf("command is required"))
		return 2
	}
	engine, err := dalle.New(dalle.Config{
		DataDir: config.dataDir,
		Provider: dalle.ProviderConfig{
			BaseURL: config.providerBaseURL,
		},
	})
	if err != nil {
		writeError(config.stderr, err)
		return 1
	}
	if err := dispatch(engine, remaining, config); err != nil {
		writeError(config.stderr, err)
		return exitCode(err)
	}
	return 0
}

type globalFlags struct {
	dataDir         string
	providerBaseURL string
}

func parseGlobalFlags(args []string) (globalFlags, []string, error) {
	var global globalFlags
	remaining := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--data-dir":
			index++
			if index >= len(args) {
				return globalFlags{}, nil, fmt.Errorf("--data-dir requires a value")
			}
			global.dataDir = args[index]
		case strings.HasPrefix(arg, "--data-dir="):
			global.dataDir = strings.TrimPrefix(arg, "--data-dir=")
		case arg == "--provider-base-url":
			index++
			if index >= len(args) {
				return globalFlags{}, nil, fmt.Errorf("--provider-base-url requires a value")
			}
			global.providerBaseURL = args[index]
		case strings.HasPrefix(arg, "--provider-base-url="):
			global.providerBaseURL = strings.TrimPrefix(arg, "--provider-base-url=")
		default:
			remaining = append(remaining, arg)
		}
	}
	return global, remaining, nil
}

func dispatch(engine *dalle.Engine, args []string, config cliConfig) error {
	switch args[0] {
	case "preview":
		return runPreview(engine, args[1:], config.stdout)
	case "generate":
		return runGenerate(engine, args[1:], config.stdout)
	case "images":
		return runImages(engine, args[1:], config.stdout)
	case "series":
		return runSeries(engine, args[1:], config)
	case "databases":
		return runDatabases(engine, args[1:], config.stdout)
	case "validate":
		if err := engine.Validate(); err != nil {
			return err
		}
		return writeJSON(config.stdout, map[string]bool{"valid": true})
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runPreview(engine *dalle.Engine, args []string, stdout io.Writer) error {
	request, err := parseGenerateRequest("preview", args)
	if err != nil {
		return err
	}
	result, err := engine.Preview(request)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runGenerate(engine *dalle.Engine, args []string, stdout io.Writer) error {
	request, err := parseGenerateRequest("generate", args)
	if err != nil {
		return err
	}
	result, err := engine.Generate(request)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func parseGenerateRequest(name string, args []string) (dalle.GenerateRequest, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	request := dalle.GenerateRequest{}
	flags.StringVar(&request.Input, "input", "", "source input")
	flags.StringVar(&request.Seed, "seed", "", "seed")
	flags.StringVar(&request.Series, "series", "", "series")
	flags.StringVar(&request.Recipe, "recipe", "", "recipe")
	flags.BoolVar(&request.Enhance, "enhance", false, "enhance prompt")
	flags.BoolVar(&request.Image, "image", false, "generate image")
	flags.BoolVar(&request.Annotate, "annotate", false, "annotate generated image")
	flags.BoolVar(&request.Force, "force", false, "ignore compatible cached metadata")
	if err := flags.Parse(reorderFlagArgs(args, map[string]bool{
		"input":    true,
		"seed":     true,
		"series":   true,
		"recipe":   true,
		"enhance":  false,
		"image":    false,
		"annotate": false,
		"force":    false,
	})); err != nil {
		return dalle.GenerateRequest{}, err
	}
	if request.Input == "" && flags.NArg() > 0 {
		request.Input = strings.Join(flags.Args(), " ")
	}
	return request, nil
}

func runImages(engine *dalle.Engine, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("images subcommand is required")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("images list", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		filter := dalle.ImageFilter{}
		flags.StringVar(&filter.Series, "series", "", "series filter")
		if err := flags.Parse(reorderFlagArgs(args[1:], map[string]bool{"series": true})); err != nil {
			return err
		}
		records, err := engine.ListImages(filter)
		if err != nil {
			return err
		}
		return writeJSON(stdout, records)
	case "show":
		id, err := requiredArg("images show", args[1:], "image ID")
		if err != nil {
			return err
		}
		record, err := engine.GetImage(id)
		if err != nil {
			return err
		}
		return writeJSON(stdout, record)
	case "export":
		return runImagesExport(engine, args[1:], stdout)
	default:
		return fmt.Errorf("unknown images subcommand %q", args[0])
	}
}

func runImagesExport(engine *dalle.Engine, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("images export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := dalle.ExportImageOptions{}
	flags.StringVar(&options.Dir, "dir", "", "export directory")
	flags.BoolVar(&options.IncludePrompt, "prompt", false, "export prompt")
	flags.BoolVar(&options.IncludeData, "data", false, "export data prompt")
	flags.BoolVar(&options.IncludeTitle, "title", false, "export title prompt")
	flags.BoolVar(&options.IncludeTerse, "terse", false, "export terse prompt")
	flags.BoolVar(&options.IncludeEnhanced, "enhanced", false, "export enhanced prompt")
	flags.BoolVar(&options.IncludeTechnical, "technical", false, "export technical prompt")
	if err := flags.Parse(reorderFlagArgs(args, map[string]bool{
		"dir":       true,
		"prompt":    false,
		"data":      false,
		"title":     false,
		"terse":     false,
		"enhanced":  false,
		"technical": false,
	})); err != nil {
		return err
	}
	id, err := requiredArg("images export", flags.Args(), "image ID")
	if err != nil {
		return err
	}
	result, err := engine.ExportImage(id, options)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runSeries(engine *dalle.Engine, args []string, config cliConfig) error {
	if len(args) == 0 {
		return fmt.Errorf("series subcommand is required")
	}
	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("series list", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		filter := dalle.SeriesFilter{}
		flags.BoolVar(&filter.IncludeHidden, "include-hidden", false, "include hidden series")
		flags.BoolVar(&filter.OnlyHidden, "only-hidden", false, "only hidden series")
		if err := flags.Parse(reorderFlagArgs(args[1:], map[string]bool{"include-hidden": false, "only-hidden": false})); err != nil {
			return err
		}
		series, err := engine.ListSeries(filter)
		if err != nil {
			return err
		}
		return writeJSON(config.stdout, series)
	case "show":
		name, err := requiredArg("series show", args[1:], "series name")
		if err != nil {
			return err
		}
		series, err := engine.GetSeries(name)
		if err != nil {
			return err
		}
		return writeJSON(config.stdout, series)
	case "save":
		return runSeriesSave(engine, args[1:], config)
	case "hide":
		return runSeriesSetHidden(engine, args[1:], config.stdout, true, "series hide")
	case "restore":
		return runSeriesSetHidden(engine, args[1:], config.stdout, false, "series restore")
	case "hidden":
		return runSeriesHidden(engine, args[1:], config.stdout)
	default:
		return fmt.Errorf("unknown series subcommand %q", args[0])
	}
}

func runSeriesSave(engine *dalle.Engine, args []string, config cliConfig) error {
	flags := flag.NewFlagSet("series save", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	series := dalle.Series{}
	jsonInput := ""
	flags.StringVar(&series.Suffix, "suffix", "", "series suffix")
	flags.IntVar(&series.Last, "last", 0, "last index")
	flags.StringVar(&series.Purpose, "purpose", "", "purpose")
	flags.StringVar(&jsonInput, "json", "", "JSON series document, or - for stdin")
	if err := flags.Parse(reorderFlagArgs(args, map[string]bool{
		"suffix":  true,
		"last":    true,
		"purpose": true,
		"json":    true,
	})); err != nil {
		return err
	}
	if jsonInput != "" {
		contents := []byte(jsonInput)
		if jsonInput == "-" {
			read, err := io.ReadAll(config.stdin)
			if err != nil {
				return err
			}
			contents = read
		}
		if err := json.Unmarshal(contents, &series); err != nil {
			return err
		}
	}
	if series.Suffix == "" && flags.NArg() > 0 {
		series.Suffix = strings.Join(flags.Args(), " ")
	}
	saved, err := engine.SaveSeries(series)
	if err != nil {
		return err
	}
	return writeJSON(config.stdout, saved)
}

func runSeriesSetHidden(engine *dalle.Engine, args []string, stdout io.Writer, hidden bool, command string) error {
	name, err := requiredArg(command, args, "series name")
	if err != nil {
		return err
	}
	series, err := engine.SetSeriesHidden(name, hidden)
	if err != nil {
		return err
	}
	return writeJSON(stdout, series)
}

func runSeriesHidden(engine *dalle.Engine, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("series hidden", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	hidden := false
	flags.BoolVar(&hidden, "hidden", false, "hidden state")
	if err := flags.Parse(reorderFlagArgs(args, map[string]bool{"hidden": false})); err != nil {
		return err
	}
	name, err := requiredArg("series hidden", flags.Args(), "series name")
	if err != nil {
		return err
	}
	series, err := engine.SetSeriesHidden(name, hidden)
	if err != nil {
		return err
	}
	return writeJSON(stdout, series)
}

func runDatabases(engine *dalle.Engine, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("databases subcommand is required")
	}
	switch args[0] {
	case "list":
		archives, err := engine.ListDatabaseArchives()
		if err != nil {
			return err
		}
		return writeJSON(stdout, archives)
	case "show":
		version, err := requiredArg("databases show", args[1:], "database version")
		if err != nil {
			return err
		}
		archive, err := engine.GetDatabaseArchive(version)
		if err != nil {
			return err
		}
		return writeJSON(stdout, archive)
	default:
		return fmt.Errorf("unknown databases subcommand %q", args[0])
	}
}

func requiredArg(command string, args []string, name string) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("%s requires %s", command, name)
	}
	return args[0], nil
}

func reorderFlagArgs(args []string, flagsWithValues map[string]bool) []string {
	flagArgs := []string{}
	positionArgs := []string{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "--") || arg == "--" {
			positionArgs = append(positionArgs, arg)
			continue
		}
		nameValue := strings.TrimPrefix(arg, "--")
		name := nameValue
		if before, _, found := strings.Cut(nameValue, "="); found {
			name = before
		}
		requiresValue, known := flagsWithValues[name]
		if !known {
			positionArgs = append(positionArgs, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		if requiresValue && !strings.Contains(arg, "=") && index+1 < len(args) {
			index++
			flagArgs = append(flagArgs, args[index])
		}
	}
	return append(flagArgs, positionArgs...)
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeError(stderr io.Writer, err error) {
	code := dalle.ErrorCodeOf(err)
	if code == "" {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(stderr, "error: %s: %v\n", code, err)
}

func exitCode(err error) int {
	switch dalle.ErrorCodeOf(err) {
	case "":
		return 2
	case dalle.ErrInvalidInput, dalle.ErrSeriesInvalid, dalle.ErrSeriesNotFound, dalle.ErrArtifactMissing:
		return 2
	default:
		return 1
	}
}
