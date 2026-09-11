package main

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestRenderDefDisambiguatesEnumValuesWithTheSameGoName(t *testing.T) {
	var body strings.Builder
	g := generator{}
	g.renderDef(&body, "ElicitationMode", &schema{
		Type: json.RawMessage(`"string"`),
		Enum: []any{"openai/form", "openaiForm"},
	}, &[]inlineType{}, map[string]bool{})

	source := "package generated\n\n" + body.String()
	if _, err := parser.ParseFile(token.NewFileSet(), "protocol.gen.go", source, parser.AllErrors); err != nil {
		t.Fatalf("generated enum does not compile: %v\n%s", err, source)
	}
	if !strings.Contains(source, `ElicitationModeOpenaiForm ElicitationMode = "openai/form"`) ||
		!strings.Contains(source, `ElicitationModeOpenaiForm_ ElicitationMode = "openaiForm"`) {
		t.Fatalf("colliding enum values were not preserved with unique names:\n%s", source)
	}
}

func TestRenderFlattenedUnionDisambiguatesTagValuesWithTheSameGoName(t *testing.T) {
	stringType := json.RawMessage(`"string"`)
	union := &schema{OneOf: []*schema{
		{Properties: map[string]*schema{"mode": {Type: stringType, Enum: []any{"openai/form"}}}, Required: []string{"mode"}},
		{Properties: map[string]*schema{"mode": {Type: stringType, Enum: []any{"openaiForm"}}}, Required: []string{"mode"}},
	}}

	var body strings.Builder
	g := generator{}
	g.renderFlattenedUnion(&body, "Elicitation", union, "mode", &[]inlineType{}, map[string]bool{})

	source := "package generated\n\n" + body.String()
	if _, err := parser.ParseFile(token.NewFileSet(), "protocol.gen.go", source, parser.AllErrors); err != nil {
		t.Fatalf("generated union discriminator does not compile: %v\n%s", err, source)
	}
	if !strings.Contains(source, `ElicitationTypeOpenaiForm ElicitationType = "openai/form"`) ||
		!strings.Contains(source, `ElicitationTypeOpenaiForm_ ElicitationType = "openaiForm"`) {
		t.Fatalf("colliding discriminator values were not preserved with unique names:\n%s", source)
	}
}
