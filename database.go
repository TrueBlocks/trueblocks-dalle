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

	if s, err := ctx.loadSeries(filter); err != nil {
		return err
	} else {
		ctx.Series = s
	}
	logger.InfoG("db.series.reload", "series", ctx.Series.Suffix)

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
	logger.InfoG("db.databases.reload", "count", len(DatabaseNames))
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
