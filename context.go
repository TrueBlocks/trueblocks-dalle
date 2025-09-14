package dalle

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/model"
	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/prompt"
	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/utils"
)

// Context holds templates, series, dbs, and cache for prompt generation.
type Context struct {
	Series         Series
	Databases      map[string][]string
	DalleCache     map[string]*model.DalleDress
	CacheMutex     sync.Mutex
	promptTemplate *template.Template
	dataTemplate   *template.Template
	titleTemplate  *template.Template
	terseTemplate  *template.Template
	authorTemplate *template.Template
}

func NewContext() *Context {
	cm := GetCacheManager()

	// Load database cache only
	if err := cm.LoadOrBuild(); err != nil {
		logger.Error("Failed to initialize caches:", err)
	}

	ctx := Context{
		promptTemplate: prompt.PromptTemplate,
		dataTemplate:   prompt.DataTemplate,
		titleTemplate:  prompt.TitleTemplate,
		terseTemplate:  prompt.TerseTemplate,
		authorTemplate: prompt.AuthorTemplate,
		Series:         Series{},
		Databases:      make(map[string][]string),
		DalleCache:     make(map[string]*model.DalleDress),
	}

	if err := ctx.ReloadDatabases("empty"); err != nil {
		logger.Error("error reloading databases:", err)
	}
	return &ctx
}

var saveMutex sync.Mutex

// reportOn logs and saves generated prompt data for a given address and location.
func (ctx *Context) reportOn(dd *model.DalleDress, addr, loc, ft, value string) {
	_ = addr
	path := filepath.Join(OutputDir(), strings.ToLower(loc))

	saveMutex.Lock()
	defer saveMutex.Unlock()
	_ = file.EstablishFolder(path)
	_ = file.StringToAsciiFile(filepath.Join(path, dd.FileName+"."+ft), value)
}

// MakeDalleDress builds or retrieves a DalleDress for the given address using the context's templates, series, dbs, and cache.
func (ctx *Context) MakeDalleDress(addressIn string) (*model.DalleDress, error) {
	ctx.CacheMutex.Lock()
	defer ctx.CacheMutex.Unlock()
	if ctx.DalleCache[addressIn] != nil {
		return ctx.DalleCache[addressIn], nil
	}

	address := addressIn
	// ENS resolution should be handled outside, but you can add it here if needed

	parts := strings.Split(address, ",")
	seed := parts[0] + utils.Reverse(parts[0])
	if len(seed) < 66 {
		return nil, fmt.Errorf("seed length is less than 66")
	}
	if strings.HasPrefix(seed, "0x") {
		seed = seed[2:66]
	}

	fn := utils.ValidFilename(address)
	if ctx.DalleCache[fn] != nil {
		return ctx.DalleCache[fn], nil
	}

	dd := model.DalleDress{
		Original:        addressIn,
		FileName:        fn,
		Seed:            seed,
		AttribMap:       make(map[string]prompt.Attribute),
		SeedChunks:      []string{},
		SelectedTokens:  []string{},
		SelectedRecords: []string{},
		Attribs:         []prompt.Attribute{},
		Series:          ctx.Series.Suffix,
	}

	// Generate attributes from the seed. We cap the number of attributes to the number of
	// configured databases (DatabaseNames) and carefully guard slice bounds so we never
	// index past the seed or database lists. The original logic could overrun both the
	// seed slicing (i+6) and the database name list when the seed was long enough to
	// create more than len(DatabaseNames) attributes.
	maxAttribs := len(prompt.DatabaseNames)
	cnt := 0
	for i := 0; i+6 <= len(dd.Seed) && cnt < maxAttribs; i += 8 {
		attr := prompt.NewAttribute(ctx.Databases, cnt, dd.Seed[i:i+6])
		dd.Attribs = append(dd.Attribs, attr)
		dd.AttribMap[attr.Name] = attr
		dd.SeedChunks = append(dd.SeedChunks, attr.Value)
		dd.SelectedTokens = append(dd.SelectedTokens, attr.Name)
		dd.SelectedRecords = append(dd.SelectedRecords, attr.Value)
		cnt++
		if cnt < maxAttribs && i+4+6 <= len(dd.Seed) {
			attr = prompt.NewAttribute(ctx.Databases, cnt, dd.Seed[i+4:i+4+6])
			dd.Attribs = append(dd.Attribs, attr)
			dd.AttribMap[attr.Name] = attr
			dd.SeedChunks = append(dd.SeedChunks, attr.Value)
			dd.SelectedTokens = append(dd.SelectedTokens, attr.Name)
			dd.SelectedRecords = append(dd.SelectedRecords, attr.Value)
			cnt++
		}
	}

	suff := ctx.Series.Suffix
	dd.DataPrompt, _ = dd.ExecuteTemplate(ctx.dataTemplate, nil)
	ctx.reportOn(&dd, addressIn, filepath.Join(suff, "data"), "txt", dd.DataPrompt)
	dd.TitlePrompt, _ = dd.ExecuteTemplate(ctx.titleTemplate, nil)
	ctx.reportOn(&dd, addressIn, filepath.Join(suff, "title"), "txt", dd.TitlePrompt)
	dd.TersePrompt, _ = dd.ExecuteTemplate(ctx.terseTemplate, nil)
	ctx.reportOn(&dd, addressIn, filepath.Join(suff, "terse"), "txt", dd.TersePrompt)
	dd.Prompt, _ = dd.ExecuteTemplate(ctx.promptTemplate, nil)
	ctx.reportOn(&dd, addressIn, filepath.Join(suff, "prompt"), "txt", dd.Prompt)
	fnPath := filepath.Join(OutputDir(), ctx.Series.Suffix, "enhanced", dd.FileName+".txt")
	if !file.FileExists(fnPath) {
		fnPath = filepath.Join(OutputDir(), ctx.Series.Suffix, "enhanced", dd.FileName+".txt")
	}
	dd.EnhancedPrompt = ""
	if file.FileExists(fnPath) {
		dd.EnhancedPrompt = file.AsciiFileToString(fnPath)
	}

	ctx.DalleCache[dd.FileName] = &dd
	ctx.DalleCache[addressIn] = &dd
	if dd.Series != ctx.Series.Suffix {
		logger.Error("MakeDalleDress:seriesMismatch", addressIn, "series", dd.Series, "loaded", ctx.Series.Suffix)
	}
	return &dd, nil
}

// GetPrompt returns the generated prompt for the given address.
func (ctx *Context) GetPrompt(addr string) string {
	if dd, err := ctx.MakeDalleDress(addr); err != nil {
		return err.Error()
	} else {
		return dd.Prompt
	}
}

// GetEnhanced returns the enhanced prompt for the given address.
func (ctx *Context) GetEnhanced(addr string) string {
	if dd, err := ctx.MakeDalleDress(addr); err != nil {
		return err.Error()
	} else {
		return dd.EnhancedPrompt
	}
}

// Save generates and saves prompt data for the given address.
func (ctx *Context) Save(addr string) bool {
	if dd, err := ctx.MakeDalleDress(addr); err != nil {
		return false
	} else {
		ctx.reportOn(dd, addr, filepath.Join(ctx.Series.Suffix, "selector"), "json", dd.String())
		return true
	}
}

// GenerateEnhanced generates an enhanced prompt for the given address.
func (ctx *Context) GenerateEnhanced(addr string) (string, error) {
	if dd, err := ctx.MakeDalleDress(addr); err != nil {
		return err.Error(), err
	} else {
		authorType, _ := dd.ExecuteTemplate(ctx.authorTemplate, nil)
		if dd.EnhancedPrompt, err = prompt.EnhancePrompt(ctx.GetPrompt(addr), authorType); err != nil {
			logger.Error("EnhancePrompt error:", err)
			return "", err
		}
		msg := " DO NOT PUT TEXT IN THE IMAGE. "
		dd.EnhancedPrompt = msg + dd.EnhancedPrompt + msg
		return dd.EnhancedPrompt, nil
	}
}

// GenerateImage generates an image for the given address.
func (ctx *Context) GenerateImage(addr string) (string, error) {
	if dd, err := ctx.MakeDalleDress(addr); err != nil {
		return err.Error(), err
	} else {
		suff := ctx.Series.Suffix
		if ep, eperr := ctx.GenerateEnhanced(addr); eperr != nil {
			return eperr.Error(), eperr
		} else {
			dd.EnhancedPrompt = ep
		}
		ctx.reportOn(dd, addr, filepath.Join(suff, "enhanced"), "txt", dd.EnhancedPrompt)
		_ = ctx.Save(addr)
		imageData := ImageData{
			TitlePrompt:    dd.TitlePrompt,
			TersePrompt:    dd.TersePrompt,
			EnhancedPrompt: dd.EnhancedPrompt,
			SeriesName:     ctx.Series.Suffix,
			Filename:       dd.FileName,
			Series:         ctx.Series.Suffix,
			Address:        addr,
		}
		// Transition to image_prep prior to network operations if progress run exists
		progressMgr.Transition(ctx.Series.Suffix, addr, PhaseImagePrep)
		outputPath := filepath.Join(OutputDir(), imageData.SeriesName, "generated")
		if err := RequestImage(outputPath, &imageData); err != nil {
			return err.Error(), err
		}
		return dd.EnhancedPrompt, nil
	}
}
