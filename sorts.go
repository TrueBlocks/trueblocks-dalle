package dalle

import (
	"sort"
	"strings"

	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/model"
	sdk "github.com/TrueBlocks/trueblocks-sdk/v5"
)

// SortDalleDress sorts in place based on field in spec
func SortDalleDress(items []model.DalleDress, sortSpec sdk.SortSpec) error {
	if len(items) < 2 || len(sortSpec.Fields) == 0 {
		return nil
	}
	if len(sortSpec.Order) == 0 {
		sortSpec.Order = append(sortSpec.Order, sdk.Asc)
	}
	field := sortSpec.Fields[0]
	asc := sortSpec.Order[0] == sdk.Asc
	cmp := func(i, j int) bool { return true }
	switch strings.ToLower(field) {
	case "original":
		cmp = func(i, j int) bool { return items[i].Original < items[j].Original }
	case "filename":
		cmp = func(i, j int) bool { return items[i].FileName < items[j].FileName }
	case "seed":
		cmp = func(i, j int) bool { return items[i].Seed < items[j].Seed }
	case "prompt":
		cmp = func(i, j int) bool { return items[i].Prompt < items[j].Prompt }
	case "dataprompt":
		cmp = func(i, j int) bool { return items[i].DataPrompt < items[j].DataPrompt }
	case "titleprompt":
		cmp = func(i, j int) bool { return items[i].TitlePrompt < items[j].TitlePrompt }
	case "terseprompt":
		cmp = func(i, j int) bool { return items[i].TersePrompt < items[j].TersePrompt }
	case "enhancedprompt":
		cmp = func(i, j int) bool { return items[i].EnhancedPrompt < items[j].EnhancedPrompt }
	case "attribs":
		cmp = func(i, j int) bool { return len(items[i].Attribs) < len(items[j].Attribs) }
	case "seedchunks":
		cmp = func(i, j int) bool { return len(items[i].SeedChunks) < len(items[j].SeedChunks) }
	case "selectedtokens":
		cmp = func(i, j int) bool { return len(items[i].SelectedTokens) < len(items[j].SelectedTokens) }
	case "selectedrecords":
		cmp = func(i, j int) bool { return len(items[i].SelectedRecords) < len(items[j].SelectedRecords) }
	case "imageurl":
		cmp = func(i, j int) bool { return items[i].ImageURL < items[j].ImageURL }
	case "generatedpath":
		cmp = func(i, j int) bool { return items[i].GeneratedPath < items[j].GeneratedPath }
	case "annotatedpath":
		cmp = func(i, j int) bool { return items[i].AnnotatedPath < items[j].AnnotatedPath }
	case "downloadmode":
		cmp = func(i, j int) bool { return items[i].DownloadMode < items[j].DownloadMode }
	case "ipfshash":
		cmp = func(i, j int) bool { return items[i].IPFSHash < items[j].IPFSHash }
	case "cachehit":
		cmp = func(i, j int) bool { return items[i].CacheHit }
	case "completed":
		cmp = func(i, j int) bool { return items[i].Completed }
	case "series":
		cmp = func(i, j int) bool { return items[i].Series < items[j].Series }
	default:
		cmp = func(i, j int) bool { return items[i].Original < items[j].Original }
	}
	sort.SliceStable(items, func(i, j int) bool {
		if asc {
			return cmp(i, j)
		}
		return !cmp(i, j)
	})
	return nil
}

// SortSeries sorts in place based on field in spec (suffix, modifiedAt, last)
func SortSeries(items []Series, sortSpec sdk.SortSpec) error {
	if len(items) < 2 || len(sortSpec.Fields) == 0 {
		return nil
	}
	if len(sortSpec.Order) == 0 {
		sortSpec.Order = append(sortSpec.Order, sdk.Asc)
	}
	field := sortSpec.Fields[0]
	asc := sortSpec.Order[0] == sdk.Asc
	cmp := func(i, j int) bool { return true }
	switch strings.ToLower(field) {
	case "suffix":
		cmp = func(i, j int) bool { return strings.Compare(items[i].Suffix, items[j].Suffix) < 0 }
	case "modifiedat":
		cmp = func(i, j int) bool { return items[i].ModifiedAt < items[j].ModifiedAt }
	case "last":
		cmp = func(i, j int) bool { return items[i].Last < items[j].Last }
	default:
		cmp = func(i, j int) bool { return strings.Compare(items[i].Suffix, items[j].Suffix) < 0 }
	}
	sort.SliceStable(items, func(i, j int) bool {
		if asc {
			return cmp(i, j)
		}
		return !cmp(i, j)
	})
	return nil
}

// SortDatabases sorts in place based on field in spec
func SortDatabases(items []model.Database, sortSpec sdk.SortSpec) error {
	if len(items) < 2 || len(sortSpec.Fields) == 0 {
		return nil
	}
	if len(sortSpec.Order) == 0 {
		sortSpec.Order = append(sortSpec.Order, sdk.Asc)
	}
	field := sortSpec.Fields[0]
	asc := sortSpec.Order[0] == sdk.Asc
	cmp := func(i, j int) bool { return true }
	switch strings.ToLower(field) {
	case "id":
		cmp = func(i, j int) bool { return items[i].ID < items[j].ID }
	case "name":
		cmp = func(i, j int) bool { return items[i].Name < items[j].Name }
	default:
		cmp = func(i, j int) bool { return items[i].Name < items[j].Name }
	}
	sort.SliceStable(items, func(i, j int) bool {
		if asc {
			return cmp(i, j)
		}
		return !cmp(i, j)
	})
	return nil
}
