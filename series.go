package dalle

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/storage"
)

// SeriesSource indicates whether a series is built into the binary or user-created.
type SeriesSource string

const (
	// SeriesSourceBuiltin marks a series that ships with the application binary.
	SeriesSourceBuiltin SeriesSource = "builtin"
	// SeriesSourceUser marks a series created by the end user.
	SeriesSourceUser SeriesSource = "user"
)

type SeriesModel struct {
	Data  map[string]any `json:"data"`
	Order []string       `json:"order"`
}

// Series represents a collection of prompt attributes and their values.
type Series struct {
	Last         int          `json:"last,omitempty"`
	Suffix       string       `json:"suffix"`
	Purpose      string       `json:"purpose,omitempty"`
	Deleted      bool         `json:"deleted,omitempty"`
	Adverbs      []string     `json:"adverbs"`
	Adjectives   []string     `json:"adjectives"`
	Nouns        []string     `json:"nouns"`
	Emotions     []string     `json:"emotions"`
	Occupations  []string     `json:"occupations"`
	Actions      []string     `json:"actions"`
	Artstyles    []string     `json:"artstyles"`
	Litstyles    []string     `json:"litstyles"`
	Colors       []string     `json:"colors"`
	Viewpoints   []string     `json:"viewpoints"`
	Gazes        []string     `json:"gazes"`
	Backstyles   []string     `json:"backstyles"`
	Compositions []string     `json:"compositions"`
	ColorLimit   string       `json:"colorLimit,omitempty"`
	ModifiedAt   string       `json:"modifiedAt,omitempty"`
	Version      string       `json:"version,omitempty"`
	Source       SeriesSource `json:"source,omitempty"`
}

func (s *Series) Model(chain, format string, verbose bool, extraOpts map[string]any) SeriesModel {
	return SeriesModel{
		Data: map[string]any{
			"suffix":       s.Suffix,
			"purpose":      s.Purpose,
			"last":         s.Last,
			"deleted":      s.Deleted,
			"modifiedAt":   s.ModifiedAt,
			"adverbs":      s.Adverbs,
			"adjectives":   s.Adjectives,
			"nouns":        s.Nouns,
			"emotions":     s.Emotions,
			"occupations":  s.Occupations,
			"actions":      s.Actions,
			"artstyles":    s.Artstyles,
			"litstyles":    s.Litstyles,
			"colors":       s.Colors,
			"viewpoints":   s.Viewpoints,
			"gazes":        s.Gazes,
			"backstyles":   s.Backstyles,
			"compositions": s.Compositions,
			"colorLimit":   s.ColorLimit,
			"version":      s.Version,
			"source":       string(s.Source),
		},
		Order: []string{"suffix", "purpose", "last", "deleted", "modifiedAt", "adverbs", "adjectives", "nouns", "emotions", "occupations", "actions", "artstyles", "litstyles", "colors", "viewpoints", "gazes", "backstyles", "compositions", "version", "source"},
	}
}

// String returns the JSON representation of the Series.
func (s *Series) String() string {
	bytes, _ := json.MarshalIndent(s, "", "  ")
	return string(bytes)
}

// SaveSeries saves the Series to a file with the given filename and last index.
func (s *Series) SaveSeries(series string, last int) error {
	ss := s
	ss.Last = last
	target := filepath.Join(storage.UserSeriesDir(), series+".json") // creates the folder
	return writeTextFile(target, ss.String())
}

// GetFilter returns a string slice for the given field name in the Series.
func (s *Series) GetFilter(fieldName string) ([]string, error) {
	reflectedT := reflect.ValueOf(s)
	field := reflect.Indirect(reflectedT).FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %s not valid", fieldName)
	}
	if field.Kind() != reflect.Slice {
		return nil, fmt.Errorf("field %s not a slice", fieldName)
	}
	if field.Type().Elem().Kind() != reflect.String {
		return nil, fmt.Errorf("field %s not a string slice", fieldName)
	}
	return field.Interface().([]string), nil
}
