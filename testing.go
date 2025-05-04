// testing.go
// Centralized test helpers and mocks for trueblocks-dalle
package dalle

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

// --- File operation mocks (used in context_test.go, database_test.go, series_test.go) ---
var (
	MockEstablishFolderCalled   bool
	MockStringToAsciiFileCalled bool
	MockFileExistsCalled        bool
	MockAsciiFileToStringCalled bool
)

func MockEstablishFolder(_ string) error {
	MockEstablishFolderCalled = true
	return nil
}

func MockStringToAsciiFile(_, _ string) error {
	MockStringToAsciiFileCalled = true
	return nil
}

func MockFileExists(_ string) bool {
	MockFileExistsCalled = true
	return false
}

func MockAsciiFileToString(_ string) string {
	MockAsciiFileToStringCalled = true
	return ""
}

func ResetFileMocks() {
	MockEstablishFolderCalled = false
	MockStringToAsciiFileCalled = false
	MockFileExistsCalled = false
	MockAsciiFileToStringCalled = false
}

// --- mockFileOps (from series_test.go) ---
type MockFileOps struct {
	EstablishCalled     bool
	StringToAsciiCalled bool
	LastFn              string
	LastContent         string
	EstablishErr        error
	StringToAsciiErr    error
}

func (m *MockFileOps) EstablishFolder(path string) error {
	m.EstablishCalled = true
	return m.EstablishErr
}

func (m *MockFileOps) StringToAsciiFile(fn, content string) error {
	m.StringToAsciiCalled = true
	m.LastFn = fn
	m.LastContent = content
	return m.StringToAsciiErr
}

// --- HTTP and template mocks (from prompt_test.go) ---
type MockRoundTripper struct {
	Resp *http.Response
	Err  error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Resp, m.Err
}

type BadReader struct{}

func (BadReader) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (BadReader) Close() error             { return nil }

// --- Minimal mock for template execution (from prompt_test.go) ---
type MockDress struct{}

func (MockDress) LitPrompt(bool) string     { return "LitPrompt" }
func (MockDress) Adverb(bool) string        { return "Adverb" }
func (MockDress) Adjective(bool) string     { return "Adjective" }
func (MockDress) Noun(bool) string          { return "Noun" }
func (MockDress) Emotion(bool) string       { return "Emotion" }
func (MockDress) Occupation(bool) string    { return "Occupation" }
func (MockDress) Action(bool) string        { return "Action" }
func (MockDress) ArtStyle(bool, int) string { return "ArtStyle" }
func (MockDress) HasLitStyle() bool         { return true }
func (MockDress) LitStyle(bool) string      { return "LitStyle" }
func (MockDress) LitStyleDescr() string     { return "LitStyleDescr" }
func (MockDress) Color(bool, int) string    { return "Color" }
func (MockDress) Orientation(bool) string   { return "Orientation" }
func (MockDress) Gaze(bool) string          { return "Gaze" }
func (MockDress) BackStyle(bool) string     { return "BackStyle" }
func (MockDress) Original() string          { return "Original" }
func (MockDress) Filename() string          { return "Filename" }
func (MockDress) Seed() string              { return "Seed" }

var (
	openFile     = os.OpenFile
	annotateFunc = annotate
	httpGet      = http.Get
	System       = func(cmd string) { utils.System(cmd) }
	ioCopy       = io.Copy
)

var openaiAPIURL = "https://api.openai.com/v1/images/generations"
var fileExists = file.FileExists
var establishFolder = file.EstablishFolder
var asciiFileToString = file.AsciiFileToString
var stringToAsciiFile = file.StringToAsciiFile
var asciiFileToLines = file.AsciiFileToLines
