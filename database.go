package dalle

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

// ReloadDatabases reloads databases applying filters from the specified series suffix.
func (ctx *Context) ReloadDatabases(series string) error {
	if series == "" {
		series = "empty"
	}
	ctx.Series = Series{}
	ctx.Databases = make(map[string][]string)
	// Load requested series file if present; otherwise leave empty (unfiltered) series with provided suffix
	fn := filepath.Join(DataDir(), "series", series+".json")
	if file.FileExists(fn) {
		str := strings.TrimSpace(file.AsciiFileToString(fn))
		if len(str) > 0 {
			var s Series
			if json.Unmarshal([]byte(str), &s) == nil {
				s.Suffix = strings.Trim(strings.ReplaceAll(s.Suffix, " ", "-"), "-")
				ctx.Series = s
			}
		}
	}
	if ctx.Series.Suffix == "" {
		ctx.Series.Suffix = strings.Trim(strings.ReplaceAll(series, " ", "-"), "-")
	}
	logger.Info("Loaded series (override):", ctx.Series.Suffix)
	// Populate databases using filters
	for _, db := range DatabaseNames {
		if ctx.Databases[db] != nil {
			continue
		}
		lines, err := readDatabaseCSV(db + ".csv")
		if err != nil {
			return err
		}
		for i := range lines {
			lines[i] = strings.Replace(lines[i], "v0.1.0,", "", -1)
		}
		if len(lines) > 0 {
			lines = lines[1:]
		}
		fnUpper := strings.ToUpper(db[:1]) + db[1:]
		if filter, ferr := ctx.Series.GetFilter(fnUpper); ferr == nil && len(filter) > 0 {
			filtered := make([]string, 0, len(lines))
			for _, line := range lines {
				for _, f := range filter {
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
	logger.Info("Loaded", len(DatabaseNames), "databases (override)")
	return nil
}
