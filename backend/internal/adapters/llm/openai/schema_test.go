package openai

import (
	"reflect"
	"testing"
)

// sampleSchema mirrors every construct Waldo's real Contract and Plan schemas
// use: an object with both required and optional properties, an optional
// nested object, an optional array of scalars, a required array of objects
// with its own optionality, enums, descriptions, and the array bounds the
// strict dialect rejects. The guarantees asserted below are structural, so
// proving them here proves them for the shipped schemas too.
func sampleSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"decision", "criteria"},
		"properties": map[string]any{
			"decision": map[string]any{
				"type": "string",
				"enum": []any{"ask", "propose"},
			},
			"question": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"text"},
				"properties": map[string]any{
					"text":         map[string]any{"type": "string"},
					"alternatives": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
			},
			"constraints": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "limits the work must respect",
			},
			"criteria": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 12,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"text"},
					"properties": map[string]any{
						"text":  map[string]any{"type": "string", "maxLength": 200},
						"notes": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func TestStrictSchemaRequiresEveryPropertyAndForbidsExtras(t *testing.T) {
	// OpenAI rejects a strict schema that leaves any property out of "required"
	// or omits additionalProperties:false. Both must hold at every depth, not
	// just at the root.
	walkObjects(t, strictSchema(sampleSchema()), func(path string, node map[string]any) {
		props, ok := node["properties"].(map[string]any)
		if !ok {
			return
		}
		if extras, ok := node["additionalProperties"].(bool); !ok || extras {
			t.Errorf("%s: additionalProperties must be false, got %v", path, node["additionalProperties"])
		}
		required := map[string]bool{}
		entries, ok := node["required"].([]any)
		if !ok {
			t.Fatalf("%s: required must be a list, got %T", path, node["required"])
		}
		for _, entry := range entries {
			required[entry.(string)] = true
		}
		if len(required) != len(props) {
			t.Errorf("%s: required lists %d of %d properties", path, len(required), len(props))
		}
		for name := range props {
			if !required[name] {
				t.Errorf("%s: property %q is missing from required", path, name)
			}
		}
	})
}

func TestStrictSchemaMakesOptionalPropertiesNullable(t *testing.T) {
	// Promoting a property to required must not make it mandatory in substance.
	// An originally-optional property becomes nullable, which decodes to the
	// same Go zero value an absent key would have produced.
	strict := strictSchema(sampleSchema())
	props := strict["properties"].(map[string]any)

	for _, name := range []string{"question", "constraints"} {
		node := props[name].(map[string]any)
		if !admitsNull(node["type"]) {
			t.Errorf("optional property %q must admit null, got type %v", name, node["type"])
		}
	}
	for _, name := range []string{"decision", "criteria"} {
		node := props[name].(map[string]any)
		if admitsNull(node["type"]) {
			t.Errorf("required property %q must not admit null, got type %v", name, node["type"])
		}
	}

	// Optionality is per-object, so a nested object's own optional properties
	// get the same treatment.
	nested := props["question"].(map[string]any)["properties"].(map[string]any)
	if !admitsNull(nested["alternatives"].(map[string]any)["type"]) {
		t.Error("nested optional property alternatives must admit null")
	}
	if admitsNull(nested["text"].(map[string]any)["type"]) {
		t.Error("nested required property text must not admit null")
	}
}

func TestStrictSchemaKeepsArrayItemsNonNullable(t *testing.T) {
	// An array element is never "optional" the way a property is: the array
	// either has the element or is shorter. Widening items to null would let
	// the model emit [null] and pass.
	strict := strictSchema(sampleSchema())
	props := strict["properties"].(map[string]any)

	items := props["criteria"].(map[string]any)["items"].(map[string]any)
	if admitsNull(items["type"]) {
		t.Errorf("array items must not admit null, got %v", items["type"])
	}
	// Even under an optional array, whose own type did widen.
	optionalItems := props["constraints"].(map[string]any)["items"].(map[string]any)
	if admitsNull(optionalItems["type"]) {
		t.Errorf("items of an optional array must not admit null, got %v", optionalItems["type"])
	}
}

func TestStrictSchemaDropsKeywordsTheDialectRejects(t *testing.T) {
	// These are a 400 from the API, not a warning. The bounds they expressed
	// are enforced by the domain on the decoded value, so dropping them costs
	// nothing.
	walkObjects(t, strictSchema(sampleSchema()), func(path string, node map[string]any) {
		for keyword := range unsupportedStrictKeywords {
			if _, present := node[keyword]; present {
				t.Errorf("%s: %q survived normalization", path, keyword)
			}
		}
	})
}

func TestStrictSchemaPreservesMeaning(t *testing.T) {
	strict := strictSchema(sampleSchema())
	decision := strict["properties"].(map[string]any)["decision"].(map[string]any)

	if got := decision["enum"]; !reflect.DeepEqual(got, []any{"ask", "propose"}) {
		t.Errorf("enum was altered: %v", got)
	}
	constraints := strict["properties"].(map[string]any)["constraints"].(map[string]any)
	if got := constraints["description"]; got != "limits the work must respect" {
		t.Errorf("description was dropped or altered: %v", got)
	}
}

func TestStrictSchemaDoesNotMutateItsInput(t *testing.T) {
	// The schema builders return a fresh map per call today, but the adapter
	// must not depend on that: a caller may reasonably hold one schema and
	// send it to two providers.
	original := sampleSchema()
	before := deepCopy(original)
	strictSchema(original)
	if !reflect.DeepEqual(original, before) {
		t.Error("strictSchema mutated the caller's schema")
	}
}

func TestSchemaNameCoercesToTheAcceptedShape(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"waldo.contract", "waldo_contract"},
		{"plan draft", "plan_draft"},
		{"already-fine_1", "already-fine_1"},
		{"   ", "reply"},
		{"", "reply"},
	} {
		if got := schemaName(tc.in); got != tc.want {
			t.Errorf("schemaName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := schemaName(strings200()); len(got) > 64 {
		t.Errorf("schemaName must cap at 64 characters, got %d", len(got))
	}
}

func strings200() string {
	out := make([]byte, 200)
	for i := range out {
		out[i] = 'a'
	}
	return string(out)
}

func admitsNull(raw any) bool {
	entries, ok := raw.([]any)
	if !ok {
		return raw == "null"
	}
	for _, entry := range entries {
		if entry == "null" {
			return true
		}
	}
	return false
}

// walkObjects visits every schema object reachable from the root, so an
// invariant is checked at all depths rather than only at the top.
func walkObjects(t *testing.T, node any, check func(path string, node map[string]any)) {
	t.Helper()
	var walk func(path string, node any)
	walk = func(path string, node any) {
		switch typed := node.(type) {
		case map[string]any:
			check(path, typed)
			for key, child := range typed {
				// "properties" holds property names, not schema keywords; its
				// children are schemas, so descend one level deeper by name.
				walk(path+"."+key, child)
			}
		case []any:
			for _, child := range typed {
				walk(path, child)
			}
		}
	}
	walk("$", node)
}

func deepCopy(node any) any {
	switch typed := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = deepCopy(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, deepCopy(value))
		}
		return out
	default:
		return node
	}
}
