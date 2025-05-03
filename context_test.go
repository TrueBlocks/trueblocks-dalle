package dalle

import (
	"sync"
	"testing"
	"text/template"
)

// --- Mocks ---
var (
	mockEstablishFolderCalled   bool
	mockStringToAsciiFileCalled bool
	mockFileExistsCalled        bool
	mockAsciiFileToStringCalled bool
)

func mockEstablishFolder(_ string) error {
	mockEstablishFolderCalled = true
	return nil
}

func mockStringToAsciiFile(_, _ string) error {
	mockStringToAsciiFileCalled = true
	return nil
}

func mockFileExists(_ string) bool {
	mockFileExistsCalled = true
	return false
}

func mockAsciiFileToString(_ string) string {
	mockAsciiFileToStringCalled = true
	return ""
}

// --- Test helpers ---
func resetMocks() {
	mockEstablishFolderCalled = false
	mockStringToAsciiFileCalled = false
	mockFileExistsCalled = false
	mockAsciiFileToStringCalled = false
}

func minimalContext() *Context {
	tmpl := template.Must(template.New("x").Parse("ok"))
	return &Context{
		PromptTemplate: tmpl,
		DataTemplate:   tmpl,
		TitleTemplate:  tmpl,
		TerseTemplate:  tmpl,
		AuthorTemplate: tmpl,
		Series:         Series{Suffix: "test"},
		Databases: map[string][]string{
			"adverbs":      {"quickly,quick,fast"},
			"adjectives":   {"red,red,red"},
			"nouns":        {"cat,cat,cat"},
			"emotions":     {"happy,happy,happy,happy,happy"},
			"occupations":  {"none,none"},
			"actions":      {"run,run"},
			"artstyles":    {"modern,modern,modern"},
			"litstyles":    {"none,none"},
			"colors":       {"blue,blue"},
			"orientations": {"left,left"},
			"gazes":        {"forward,forward"},
			"backstyles":   {"plain,plain"},
		},
		DalleCache: make(map[string]*DalleDress),
		CacheMutex: sync.Mutex{},
	}
}

// --- Tests ---
func TestMakeDalleDress_ValidAndCache(t *testing.T) {
	ctx := minimalContext()
	oldEstablish := establishFolder
	oldStringToAscii := stringToAsciiFile
	oldFileExists := fileExists
	oldAsciiFileToString := asciiFileToString
	defer func() {
		establishFolder = oldEstablish
		stringToAsciiFile = oldStringToAscii
		fileExists = oldFileExists
		asciiFileToString = oldAsciiFileToString
	}()
	establishFolder = mockEstablishFolder
	stringToAsciiFile = mockStringToAsciiFile
	fileExists = mockFileExists
	asciiFileToString = mockAsciiFileToString

	resetMocks()
	addr := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	dress, err := ctx.MakeDalleDress(addr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dress == nil {
		t.Fatal("expected non-nil DalleDress")
	}
	if !mockEstablishFolderCalled || !mockStringToAsciiFileCalled {
		t.Error("expected file operations to be called")
	}
	// Test cache
	resetMocks()
	dress2, err2 := ctx.MakeDalleDress(addr)
	if err2 != nil || dress2 != dress {
		t.Error("expected cached DalleDress to be returned")
	}
}

func TestMakeDalleDress_InvalidAddress(t *testing.T) {
	ctx := minimalContext()
	addr := "short"
	_, err := ctx.MakeDalleDress(addr)
	if err == nil || err.Error() != "seed length is less than 66" {
		t.Errorf("expected seed length error, got %v", err)
	}
}

func TestGetPromptAndEnhanced(t *testing.T) {
	ctx := minimalContext()
	oldEstablish := establishFolder
	oldStringToAscii := stringToAsciiFile
	oldFileExists := fileExists
	oldAsciiFileToString := asciiFileToString
	defer func() {
		establishFolder = oldEstablish
		stringToAsciiFile = oldStringToAscii
		fileExists = oldFileExists
		asciiFileToString = oldAsciiFileToString
	}()
	establishFolder = mockEstablishFolder
	stringToAsciiFile = mockStringToAsciiFile
	fileExists = mockFileExists
	asciiFileToString = mockAsciiFileToString

	resetMocks()
	addr := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	_, _ = ctx.MakeDalleDress(addr) // populate cache
	if got := ctx.GetPrompt(addr); got == "" {
		t.Errorf("GetPrompt = %q, want non-empty", got)
	}
	if got := ctx.GetEnhanced(addr); got != "" {
		t.Errorf("GetEnhanced = %q, want empty (no enhanced prompt)", got)
	}
}

func TestSave(t *testing.T) {
	ctx := minimalContext()
	oldEstablish := establishFolder
	oldStringToAscii := stringToAsciiFile
	oldFileExists := fileExists
	oldAsciiFileToString := asciiFileToString
	defer func() {
		establishFolder = oldEstablish
		stringToAsciiFile = oldStringToAscii
		fileExists = oldFileExists
		asciiFileToString = oldAsciiFileToString
	}()
	establishFolder = mockEstablishFolder
	stringToAsciiFile = mockStringToAsciiFile
	fileExists = mockFileExists
	asciiFileToString = mockAsciiFileToString

	resetMocks()
	addr := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	_, _ = ctx.MakeDalleDress(addr)
	if !ctx.Save(addr) {
		t.Error("Save should return true on success")
	}
	if ctx.Save("short") {
		t.Error("Save should return false on error")
	}
}

// Additional tests for GenerateEnhanced and GenerateImage would require more extensive mocking and are omitted for brevity.
