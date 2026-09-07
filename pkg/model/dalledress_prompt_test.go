package model

import (
	"strings"
	"testing"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/prompt"
)

func sampleDress() *DalleDress {
	return &DalleDress{
		Original: "seed-original",
		FileName: "seed-file",
		AttribMap: map[string]prompt.Attribute{
			"noun":        {Value: "griffin,mythic griffin"},
			"emotion":     {Value: "melancholy,deep melancholy"},
			"adverb":      {Value: "quietly,very quietly"},
			"adjective":   {Value: "ancient,impossibly ancient"},
			"occupation":  {Value: "clockmaker,master clockmaker"},
			"action":      {Value: "reading,reading a letter"},
			"artStyle1":   {Value: "art-nouveau,flowing art nouveau"},
			"artStyle2":   {Value: "baroque,ornate baroque"},
			"backStyle":   {Value: "misty,misty haze"},
			"composition": {Value: "thirds,rule of thirds"},
			"color1":      {Value: "crimson,#dc143c"},
			"color2":      {Value: "gold,#ffd700"},
			"color3":      {Value: "ivory,#fffff0"},
			"gaze":        {Value: "outward,facing outward"},
			"litStyle":    {Value: "none"},
			"place":       {Value: "a workshop,a candlelit workshop"},
			"trope":       {Value: "last of his kind,the last of his kind"},
			"viewpoint":   {Value: "wide,a wide establishing shot"},
		},
	}
}

func TestImagePromptTemplateCarriesSubjectAndFraming(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	dd := sampleDress()
	out, err := prompt.ImagePrompt.Fill(dd)
	if err != nil {
		t.Fatalf("rendering the image prompt: %v", err)
	}
	for _, want := range []string{"Draw a", "human-like", "griffin", "melancholy", "Focus on the emotion"} {
		if !strings.Contains(out, want) {
			t.Errorf("image prompt is missing %q", want)
		}
	}
}

func TestTerseTemplateNamesTheArtStyle(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	dd := sampleDress()
	out, err := prompt.TersePrompt.Fill(dd)
	if err != nil {
		t.Fatalf("rendering the terse descriptor: %v", err)
	}
	for _, want := range []string{"griffin", "in the style of"} {
		if !strings.Contains(out, want) {
			t.Errorf("terse descriptor is missing %q", want)
		}
	}
}

func TestTechnicalTemplateForbidsTextInTheImage(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	dd := sampleDress()
	out, err := prompt.TechnicalPrompt.Fill(dd)
	if err != nil {
		t.Fatalf("rendering the technical block: %v", err)
	}
	if !strings.Contains(out, "Technical Specifications:") {
		t.Error("technical block is missing its header")
	}
	if !strings.Contains(out, "DO NOT PUT TEXT IN THE IMAGE.") {
		t.Error("technical block must forbid text in the image")
	}
}

func TestAuthorPromptTakesOnThePersona(t *testing.T) {
	t.Setenv("TRUEBLOCKS_DATA_DIR", t.TempDir())
	dd := sampleDress()
	dd.AttribMap["litStyle"] = prompt.Attribute{Value: "gothic,dwells on decay and dread"}
	out, err := prompt.AuthorPrompt.Fill(dd)
	if err != nil {
		t.Fatalf("rendering the author persona: %v", err)
	}
	for _, want := range []string{"award winning author", "gothic", "persona"} {
		if !strings.Contains(out, want) {
			t.Errorf("author persona is missing %q", want)
		}
	}

	dd.AttribMap["litStyle"] = prompt.Attribute{Value: "none"}
	blank, err := prompt.AuthorPrompt.Fill(dd)
	if err != nil {
		t.Fatalf("rendering the author persona without a literary style: %v", err)
	}
	if strings.TrimSpace(blank) != "" {
		t.Errorf("author persona should be empty without a literary style, got %q", blank)
	}
}
