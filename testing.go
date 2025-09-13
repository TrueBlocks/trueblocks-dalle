// testing.go
// Centralized test helpers and mocks for trueblocks-dalle
package dalle

import (
	"io"
	"net/http"
	"os"
)

var (
	openFile     = os.OpenFile
	annotateFunc = annotate
	httpGet      = http.Get
	ioCopy       = io.Copy
)

var openaiAPIURL = "https://api.openai.com/v1/images/generations"
