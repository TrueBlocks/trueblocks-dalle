package dalle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/colors"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

type ImageData struct {
	EnhancedPrompt string `json:"enhancedPrompt"`
	TersePrompt    string `json:"tersePrompt"`
	TitlePrompt    string `json:"titlePrompt"`
	SeriesName     string `json:"seriesName"`
	Filename       string `json:"filename"`
}

func RequestImage(outputPath string, imageData *ImageData) error {
	start := time.Now()
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- RequestImage:start", colors.Off)
	generated := outputPath
	_ = file.EstablishFolder(generated)
	annotated := strings.Replace(generated, "/generated", "/annotated", -1)
	_ = file.EstablishFolder(annotated)

	fn := filepath.Join(generated, fmt.Sprintf("%s.png", imageData.Filename))
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- improving the prompt...", colors.Off)

	size := "1024x1024"
	if strings.Contains(imageData.EnhancedPrompt, "horizontal") {
		size = "1792x1024"
	} else if strings.Contains(imageData.EnhancedPrompt, "vertical") {
		size = "1024x1792"
	}

	quality := "standard"
	if os.Getenv("DALLE_QUALITY") != "" {
		quality = os.Getenv("DALLE_QUALITY")
	}

	url := openaiAPIURL
	payload := dalleRequest{
		Prompt:  imageData.EnhancedPrompt,
		N:       1,
		Quality: quality,
		Style:   "vivid",
		Model:   "dall-e-3",
		Size:    size,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// No key: create a placeholder empty annotated file and return
		placeholder := filepath.Join(annotated, fmt.Sprintf("%s.png", imageData.Filename))
		_ = os.WriteFile(placeholder, []byte{}, 0600)
		logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- skipped image generation (no OPENAI_API_KEY)", colors.Off)
		logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- RequestImage:end", time.Since(start).String(), colors.Off)
		return nil
	}
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- generating the image...", colors.Off)

	imgTO := 30 * time.Second
	if v := os.Getenv("DALLESERVER_IMAGE_TIMEOUT"); v != "" {
		if d, err2 := time.ParseDuration(v); err2 == nil {
			imgTO = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), imgTO)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	reqStart := time.Now()
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- POSTing image request", colors.Off)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- image request responded in "+time.Since(reqStart).String(), colors.Off)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyStr := string(body)
	body = []byte(bodyStr)

	var dalleResp dalleResponse1
	err = json.Unmarshal(body, &dalleResp)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error: %s %d %s", resp.Status, resp.StatusCode, string(body))
	}

	if len(dalleResp.Data) == 0 {
		return fmt.Errorf("no images returned")
	}

	imageURL := dalleResp.Data[0].Url

	ctx2, cancel2 := context.WithTimeout(context.Background(), imgTO)
	defer cancel2()
	imageReq, err := http.NewRequestWithContext(ctx2, "GET", imageURL, nil)
	if err != nil {
		return err
	}
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- downloading image", colors.Off)
	imageResp, err := (&http.Client{}).Do(imageReq)
	if err != nil {
		return err
	}
	defer imageResp.Body.Close()

	os.Remove(fn)
	file, err := openFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open output file: %s", fn)
	}
	defer file.Close()

	_, err = ioCopy(file, imageResp.Body)
	if err != nil {
		return err
	}

	path, err := annotateFunc(imageData.TersePrompt, fn, "bottom", 0.2)
	if err != nil {
		return fmt.Errorf("error annotating image: %v", err)
	}
	logger.Info(colors.Cyan, imageData.Filename, colors.Green, "- image saved as", colors.White+strings.Trim(path, " "), colors.Off)
	logger.Info(colors.Cyan, imageData.Filename, colors.Yellow, "- RequestImage:end", time.Since(start).String(), colors.Off)
	if os.Getenv("TB_CMD_LINE") == "true" {
		utils.System("open " + path)
	}
	return nil
}
