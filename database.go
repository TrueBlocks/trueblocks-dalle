package dalle

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

// ReloadDatabases reloads databases applying filters from the specified series suffix.
func (ctx *Context) ReloadDatabases(filter string) error {
	ctx.Series = Series{}
	ctx.Databases = make(map[string][]string)

	if s, err := ctx.LoadSeries(); err != nil {
		return err
	} else {
		ctx.Series = s
	}
	logger.InfoG("db.series.load", "series", ctx.Series.Suffix)

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
			lines = lines[1:] // skip header
		}
		fn := strings.ToUpper(db[:1]) + db[1:]
		if filter, ferr := ctx.Series.GetFilter(fn); ferr == nil && len(filter) > 0 {
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
	logger.InfoG("db.databases.load", "count", len(DatabaseNames))
	return nil
}

func (ctx *Context) LoadSeries() (Series, error) {
	lastSeries := "five-tone-postal-protozoa" // ctx.GetSession().LastSeries
	fn := filepath.Join(DataDir(), "series", lastSeries+".json")
	str := strings.TrimSpace(file.AsciiFileToString(fn))
	logger.Info("lastSeries", lastSeries)
	if len(str) == 0 || !file.FileExists(fn) {
		logger.Info("No series found, creating a new one", fn)
		ret := Series{
			Suffix: "empty",
		}
		ret.SaveSeries(filepath.Join(DataDir(), "series"), fn, 0)
		return ret, nil
	}

	bytes := []byte(str)
	var s Series
	if err := json.Unmarshal(bytes, &s); err != nil {
		logger.Error("could not unmarshal series:", err)
		return Series{}, err
	}

	s.Suffix = strings.Trim(strings.ReplaceAll(s.Suffix, " ", "-"), "-")
	s.SaveSeries(filepath.Join(DataDir(), "series"), filepath.Join(dataDir, "series", s.Suffix+".json"), 0)
	return s, nil
}
