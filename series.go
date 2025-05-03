package dalle

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
)

type Series struct {
	Last         int      `json:"last,omitempty"`
	Suffix       string   `json:"suffix"`
	Adverbs      []string `json:"adverbs"`
	Adjectives   []string `json:"adjectives"`
	Nouns        []string `json:"nouns"`
	Emotions     []string `json:"emotions"`
	Occupations  []string `json:"occupations"`
	Actions      []string `json:"actions"`
	Artstyles    []string `json:"artstyles"`
	Litstyles    []string `json:"litstyles"`
	Colors       []string `json:"colors"`
	Orientations []string `json:"orientations"`
	Gazes        []string `json:"gazes"`
	Backstyles   []string `json:"backstyles"`
}

func (s *Series) String() string {
	bytes, _ := json.MarshalIndent(s, "", "  ")
	return string(bytes)
}

func (s *Series) SaveSeries(fn string, last int) {
	ss := s
	ss.Last = last
	_ = file.EstablishFolder("output/series")
	_ = file.StringToAsciiFile(fn, ss.String())
}

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

// func (a *App) GetSeries(baseFolder string) []utils.Series {
// 	folder := filepath.Join(baseFolder)
// 	if list, err := utils.Listing(folder); err != nil {
// 		return []utils.Series{utils.Series(err.Error())}
// 	} else {
// 		return list
// 	}
// }
