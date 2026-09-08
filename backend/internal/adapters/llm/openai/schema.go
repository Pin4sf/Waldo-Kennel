package openai

// OpenAI's strict Structured Outputs mode guarantees the reply matches the
// schema, but only accepts a restricted dialect of JSON Schema. Kennel's
// reasoning schemas are authored once and shared by every provider, so rather
// than weaken them to the narrowest common denominator we translate them here,
// at the edge that has the constraint.
//
// The translation is lossy in one direction only: it drops advisory bounds
// that the strict dialect rejects. Nothing is lost in practice, because the
// schema is a hint and the Go control plane is the authority — every reply is
// re-validated against the domain before it can become a ContractRevision or a
// PlanRevision.

// unsupportedStrictKeywords are validation keywords outside OpenAI's strict
// subset. Sending one is a 400, not a soft warning, so they are removed. Each
// is a bound the domain already enforces on the decoded value.
var unsupportedStrictKeywords = map[string]bool{
	"minItems": true, "maxItems": true, "uniqueItems": true,
	"minLength": true, "maxLength": true, "pattern": true, "format": true,
	"minimum": true, "maximum": true, "multipleOf": true,
	"exclusiveMinimum": true, "exclusiveMaximum": true,
	"minProperties": true, "maxProperties": true,
	"default": true, "examples": true,
}

// strictSchema rewrites a schema into OpenAI's strict dialect.
//
// Two rules do the work: every object must list all of its properties as
// required, and every object must forbid additional properties. An optional
// property therefore cannot be expressed by omission, so it becomes a required
// property that is allowed to be null — which decodes to exactly the same Go
// zero value (nil pointer, nil slice) an absent key would have produced.
func strictSchema(schema map[string]any) map[string]any {
	rewritten, ok := rewriteNode(schema, true).(map[string]any)
	if !ok {
		return schema
	}
	return rewritten
}

// rewriteNode returns a normalized copy, leaving the caller's schema untouched.
// required reports whether the parent declared this node mandatory; when it did
// not, the node widens to admit null.
func rewriteNode(node any, required bool) any {
	switch typed := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed)+1)
		for key, value := range typed {
			if unsupportedStrictKeywords[key] {
				continue
			}
			out[key] = value
		}

		if props, ok := out["properties"].(map[string]any); ok {
			wasRequired := requiredSet(out["required"])
			rewritten := make(map[string]any, len(props))
			names := make([]string, 0, len(props))
			for name, child := range props {
				rewritten[name] = rewriteNode(child, wasRequired[name])
				names = append(names, name)
			}
			out["properties"] = rewritten
			// Strict mode requires every property to be listed. Order does not
			// matter to the API, but a stable one keeps the prompt prefix
			// identical between calls so OpenAI's cache can hit.
			out["required"] = sortedNames(names)
			out["additionalProperties"] = false
		}

		if items, ok := out["items"]; ok {
			// Array elements are always present when the array is; an element
			// is never "optional" the way a property is.
			out["items"] = rewriteNode(items, true)
		}

		if !required {
			out["type"] = nullableType(out["type"])
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, entry := range typed {
			out = append(out, rewriteNode(entry, true))
		}
		return out
	default:
		return node
	}
}

// requiredSet reads the pre-rewrite "required" list. Anything not named there
// was optional and has to become nullable.
func requiredSet(raw any) map[string]bool {
	names := map[string]bool{}
	entries, ok := raw.([]any)
	if !ok {
		return names
	}
	for _, entry := range entries {
		if name, ok := entry.(string); ok {
			names[name] = true
		}
	}
	return names
}

// nullableType widens a type declaration to admit null, without duplicating
// null if the schema already allowed it.
func nullableType(raw any) any {
	switch typed := raw.(type) {
	case string:
		if typed == "null" {
			return typed
		}
		return []any{typed, "null"}
	case []any:
		for _, entry := range typed {
			if name, ok := entry.(string); ok && name == "null" {
				return typed
			}
		}
		return append(append([]any{}, typed...), "null")
	default:
		// No declared type, so nothing to widen and null is already admissible.
		return raw
	}
}

// sortedNames orders a small, fixed property list. An insertion sort keeps this
// dependency-free and reads as what it is.
func sortedNames(names []string) []any {
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, name)
	}
	return out
}
