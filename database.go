package dalle

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	sdk "github.com/TrueBlocks/trueblocks-sdk/v5"
)

// ReloadDatabases reloads databases applying filters from the specified series suffix.
// Now uses binary cache for improved performance while maintaining immutability.
func (ctx *Context) ReloadDatabases(filter string) error {
	ctx.Series = Series{}
	ctx.Databases = make(map[string][]string)

	if s, err := ctx.loadSeries(filter); err != nil {
		return err
	} else {
		ctx.Series = s
	}
	logger.InfoG("db.series.reload", "series", ctx.Series.Suffix)

	// Ensure cache manager is loaded
	cm := GetCacheManager()
	if err := cm.LoadOrBuild(); err != nil {
		logger.Error("Failed to load cache manager, using fallback:", err)
	}

	for _, db := range DatabaseNames {
		if ctx.Databases[db] != nil {
			continue
		}

		// Try to get database from cache first
		dbIndex, err := cm.GetDatabase(db)
		if err != nil {
			logger.Error("Failed to get database from cache, using fallback:", err)
			// Fall back to original CSV reading
			if fallbackErr := ctx.loadDatabaseFallback(db, filter); fallbackErr != nil {
				return fallbackErr
			}
			continue
		}

		// Convert database index to string slice format (for compatibility)
		lines := make([]string, 0, len(dbIndex.Records))
		for _, record := range dbIndex.Records {
			// Reconstruct CSV line from record values
			line := strings.Join(record.Values, ",")
			lines = append(lines, line)
		}

		// Apply series filters if configured
		fn := strings.ToUpper(db[:1]) + db[1:]
		if seriesFilter, ferr := ctx.Series.GetFilter(fn); ferr == nil && len(seriesFilter) > 0 {
			filtered := make([]string, 0, len(lines))
			for _, line := range lines {
				for _, f := range seriesFilter {
					if strings.Contains(line, f) {
						filtered = append(filtered, line)
						break
					}
				}
			}
			lines = filtered
		}

		if len(lines) == 0 {
			lines = append(lines, "none")
		}
		ctx.Databases[db] = lines
	}
	logger.InfoG("db.databases.reload", "count", len(DatabaseNames))
	return nil
}

// loadDatabaseFallback provides fallback to original CSV reading method
func (ctx *Context) loadDatabaseFallback(db, filter string) error {
	lines, err := readDatabaseCSV(db + ".csv")
	if err != nil {
		return err
	}

	// Remove version prefixes
	for i := range lines {
		lines[i] = strings.Replace(lines[i], "v0.1.0,", "", -1)
	}

	if len(lines) > 0 {
		lines = lines[1:] // skip header
	}

	// Apply series filters
	fn := strings.ToUpper(db[:1]) + db[1:]
	if seriesFilter, ferr := ctx.Series.GetFilter(fn); ferr == nil && len(seriesFilter) > 0 {
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			for _, f := range seriesFilter {
				if strings.Contains(line, f) {
					filtered = append(filtered, line)
					break
				}
			}
		}
		lines = filtered
	}

	if len(lines) == 0 {
		lines = append(lines, "none")
	}

	ctx.Databases[db] = lines
	return nil
}

func (ctx *Context) loadSeries(filterIn string) (Series, error) {
	logger.Info("db.load.series", "series", filterIn)
	filter := strings.ToLower(strings.Trim(strings.ReplaceAll(filterIn, " ", "-"), "-"))
	if filterIn != filter {
		logger.Info("db.load.series", "series", filterIn, "normalized", filter)
	}

	fn := filepath.Join(DataDir(), "series", filter+".json")
	str := strings.TrimSpace(file.AsciiFileToString(fn))

	ret := Series{
		Suffix: filter,
	}

	if !file.FileExists(fn) || len(str) == 0 {
		logger.Info("no series found, creating a new file", fn)
		ret.SaveSeries(filter, 0)
		return ret, nil
	}

	if err := json.Unmarshal([]byte(str), &ret); err != nil {
		logger.Error("could not unmarshal series:", err)
		return ret, err
	}

	return ret, nil
}

// SortDatabases sorts in place based on field in spec
func SortDatabases(items []Database, sortSpec sdk.SortSpec) error {
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
