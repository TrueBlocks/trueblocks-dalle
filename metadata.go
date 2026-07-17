package dalle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	MetadataVersion      = "1.0.0"
	DefaultRecipeName    = "default"
	DefaultRecipeVersion = "1.0.0"
)

type ImageMetadata struct {
	MetadataVersion string           `json:"metadataVersion"`
	ImageID         string           `json:"imageId"`
	Input           string           `json:"input"`
	Seed            string           `json:"seed"`
	Series          MetadataSeries   `json:"series"`
	Recipe          MetadataRecipe   `json:"recipe"`
	Database        MetadataDatabase `json:"database"`
	SelectedRecords []SelectedRecord `json:"selectedRecords"`
	Prompts         PromptSet        `json:"prompts"`
	Artifacts       ArtifactSet      `json:"artifacts"`
	Stages          PipelineStages   `json:"stages"`
	Status          MetadataStatus   `json:"status"`
}

type MetadataSeries struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Source string `json:"source"`
}

type MetadataRecipe struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MetadataDatabase struct {
	Version     string `json:"version"`
	ArchiveHash string `json:"archiveHash"`
}

type SelectedRecord struct {
	Attribute string `json:"attribute"`
	Database  string `json:"database"`
	RowIndex  int    `json:"rowIndex"`
	Record    string `json:"record"`
}

type PromptSet struct {
	Prompt         string `json:"prompt,omitempty"`
	DataPrompt     string `json:"dataPrompt,omitempty"`
	TitlePrompt    string `json:"titlePrompt,omitempty"`
	TersePrompt    string `json:"tersePrompt,omitempty"`
	EnhancedPrompt string `json:"enhancedPrompt,omitempty"`
}

type ArtifactSet struct {
	Generated string `json:"generated,omitempty"`
	Annotated string `json:"annotated,omitempty"`
}

type PipelineStages struct {
	Selected  StageStatus `json:"selected"`
	Prompted  StageStatus `json:"prompted"`
	Enhanced  StageStatus `json:"enhanced"`
	Generated StageStatus `json:"generated"`
	Annotated StageStatus `json:"annotated"`
}

type StageStatus struct {
	Status   string `json:"status"`
	CacheHit bool   `json:"cacheHit,omitempty"`
	Error    string `json:"error,omitempty"`
}

type MetadataStatus struct {
	Completed bool `json:"completed"`
	CacheHit  bool `json:"cacheHit"`
}

type ImageFilter struct {
	Series          string
	IncludeArchived bool
}

type ImageMetadataRecord struct {
	Path     string        `json:"path"`
	Metadata ImageMetadata `json:"metadata"`
	Archived bool          `json:"archived"`
}

func NewImageMetadata(input, seed, series string) ImageMetadata {
	metadata := ImageMetadata{
		MetadataVersion: MetadataVersion,
		Input:           input,
		Seed:            seed,
		Series: MetadataSeries{
			Name: series,
		},
		Recipe: MetadataRecipe{
			Name:    DefaultRecipeName,
			Version: DefaultRecipeVersion,
		},
	}
	metadata.ImageID = ComputeImageID(metadata)
	return metadata
}

func ComputeImageID(metadata ImageMetadata) string {
	hashInput := strings.Join([]string{
		metadata.Input,
		metadata.Seed,
		metadata.Series.Name,
		metadata.Series.Hash,
		metadata.Recipe.Name,
		metadata.Recipe.Version,
		metadata.Database.Version,
		metadata.Database.ArchiveHash,
	}, "\x00")
	digest := sha256.Sum256([]byte(hashInput))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func MetadataPath(dataDir, series, seed string) (string, error) {
	if strings.TrimSpace(series) == "" {
		return "", NewError(ErrInvalidInput, "series is required")
	}
	if strings.TrimSpace(seed) == "" {
		return "", NewError(ErrInvalidInput, "seed is required")
	}
	return filepath.Join(dataDir, "output", safePathPart(series), "metadata", safePathPart(seed)+".json"), nil
}

func ReadImageMetadata(path string) (ImageMetadata, error) {
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ImageMetadata{}, WrapError(ErrMetadataInvalid, "read image metadata", err)
	}
	var metadata ImageMetadata
	if err := json.Unmarshal(contents, &metadata); err != nil {
		return ImageMetadata{}, WrapError(ErrMetadataInvalid, "decode image metadata", err)
	}
	if err := ValidateImageMetadata(metadata); err != nil {
		return ImageMetadata{}, err
	}
	return metadata, nil
}

func WriteImageMetadata(dataDir string, metadata ImageMetadata) (string, error) {
	if metadata.MetadataVersion == "" {
		metadata.MetadataVersion = MetadataVersion
	}
	if metadata.Recipe.Name == "" {
		metadata.Recipe.Name = DefaultRecipeName
	}
	if metadata.Recipe.Version == "" {
		metadata.Recipe.Version = DefaultRecipeVersion
	}
	if metadata.ImageID == "" {
		metadata.ImageID = ComputeImageID(metadata)
	}
	if err := ValidateImageMetadata(metadata); err != nil {
		return "", err
	}
	path, err := MetadataPath(dataDir, metadata.Series.Name, metadata.Seed)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", WrapError(ErrMetadataInvalid, "create metadata directory", err)
	}
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", WrapError(ErrMetadataInvalid, "encode image metadata", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return "", WrapError(ErrMetadataInvalid, "write image metadata", err)
	}
	return path, nil
}

func ValidateImageMetadata(metadata ImageMetadata) error {
	if metadata.MetadataVersion == "" {
		return NewError(ErrMetadataInvalid, "metadata version is required")
	}
	if metadata.Input == "" {
		return NewError(ErrMetadataInvalid, "input is required")
	}
	if metadata.Seed == "" {
		return NewError(ErrMetadataInvalid, "seed is required")
	}
	if metadata.Series.Name == "" {
		return NewError(ErrMetadataInvalid, "series name is required")
	}
	if metadata.Recipe.Name == "" || metadata.Recipe.Version == "" {
		return NewError(ErrMetadataInvalid, "recipe name and version are required")
	}
	return nil
}

func ListImageMetadata(dataDir string, filter ImageFilter) ([]ImageMetadataRecord, error) {
	outputDir := filepath.Join(dataDir, "output")
	archivesDir := filepath.Join(dataDir, "archives")
	seriesFilter := strings.TrimSpace(filter.Series)
	records := []ImageMetadataRecord{}
	walkMetadataDir := func(root string, archived bool) error {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return WrapError(ErrMetadataInvalid, "inspect image metadata directory", err)
		}
		return filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if dirEntry.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}
			parentDir := filepath.Base(filepath.Dir(path))
			if parentDir != "metadata" {
				return nil
			}
			grandparentDir := filepath.Base(filepath.Dir(filepath.Dir(path)))
			isArchived := archived || grandparentDir == "archive"
			if isArchived && !filter.IncludeArchived {
				return nil
			}
			metadata, err := ReadImageMetadata(path)
			if err != nil {
				return err
			}
			if seriesFilter != "" && metadata.Series.Name != seriesFilter {
				return nil
			}
			records = append(records, ImageMetadataRecord{Path: path, Metadata: metadata, Archived: isArchived})
			return nil
		})
	}
	if err := walkMetadataDir(outputDir, false); err != nil {
		return nil, WrapError(ErrMetadataInvalid, "list image metadata", err)
	}
	if err := walkMetadataDir(archivesDir, true); err != nil {
		return nil, WrapError(ErrMetadataInvalid, "list image metadata", err)
	}
	if len(records) == 0 {
		if _, err := os.Stat(outputDir); err != nil {
			if os.IsNotExist(err) {
				return records, nil
			}
			return nil, WrapError(ErrMetadataInvalid, "inspect output directory", err)
		}
	}
	// Order by seed phrase first, then series, so that one input's variants across
	// series sit together. Sorting by Path would group by series instead, because
	// metadata lives at output/<series>/metadata/<seed>.json. Path breaks ties to
	// keep the order total and stable.
	sort.Slice(records, func(left, right int) bool {
		a, b := records[left], records[right]
		if a.Metadata.Input != b.Metadata.Input {
			return a.Metadata.Input < b.Metadata.Input
		}
		if a.Metadata.Series.Name != b.Metadata.Series.Name {
			return a.Metadata.Series.Name < b.Metadata.Series.Name
		}
		return a.Path < b.Path
	})
	return records, nil
}

func CheckRegenerationCompatibility(metadata ImageMetadata, databaseVersion, archiveHash string) error {
	if metadata.Database.Version != "" && metadata.Database.Version != databaseVersion {
		return WrapError(ErrRegenerationRefused, "database version differs", NewError(ErrDatabaseVersionUnavailable, fmt.Sprintf("metadata uses %s; current archive is %s", metadata.Database.Version, databaseVersion)))
	}
	if metadata.Database.ArchiveHash != "" && metadata.Database.ArchiveHash != archiveHash {
		return WrapError(ErrRegenerationRefused, "database archive hash differs", NewError(ErrDatabaseHashMismatch, fmt.Sprintf("metadata uses %s; current archive is %s", metadata.Database.ArchiveHash, archiveHash)))
	}
	return nil
}

func safePathPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, string(os.PathSeparator), "-")
	value = strings.ReplaceAll(value, "..", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
