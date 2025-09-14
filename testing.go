// testing.go
// Centralized test helpers and mocks for trueblocks-dalle
package dalle

import (
	"io"
	"net/http"
	"os"

	"github.com/TrueBlocks/trueblocks-dalle/v2/pkg/annotate"
)

var (
	openFile     = os.OpenFile
	annotateFunc = annotate.Annotate
	httpGet      = http.Get
	ioCopy       = io.Copy
)

var openaiAPIURL = "https://api.openai.com/v1/images/generations"
