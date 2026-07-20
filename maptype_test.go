package go2cty2go

import (
	"encoding/json"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// A map[string]any is a record — that is what JSON decoding produces — so
// its cty type must not depend on whether the current values happen to
// share a type. These cases previously alternated between map and object.
func TestAnyToCtyStringAnyMapIsAlwaysObject(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"all numbers", `{"a":1,"b":2}`},
		{"single number", `{"a":1}`},
		{"all strings", `{"a":"x","b":"y"}`},
		{"mixed", `{"a":1,"b":"x"}`},
		{"with null", `{"a":1,"b":null}`},
		{"empty", `{}`},
		{"nested homogeneous", `{"a":{"n":1}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v any
			if err := json.Unmarshal([]byte(tt.json), &v); err != nil {
				t.Fatal(err)
			}
			got, err := AnyToCty(v)
			if err != nil {
				t.Fatalf("AnyToCty() error = %v", err)
			}
			if !got.Type().IsObjectType() {
				t.Errorf("AnyToCty(%s) type = %s, want an object type", tt.json, got.Type().GoString())
			}
		})
	}
}

// A map with a concrete element type is a genuine homogeneous map and must
// stay one — the record rule applies only to map[string]any.
func TestAnyToCtyTypedMapStaysAMap(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want cty.Type
	}{
		{"map[string]string", map[string]string{"a": "x", "b": "y"}, cty.Map(cty.String)},
		{"map[string]int", map[string]int{"a": 1}, cty.Map(cty.Number)},
		{"map[string]bool", map[string]bool{"a": true}, cty.Map(cty.Bool)},
		{"empty typed map", map[string]string{}, cty.Map(cty.DynamicPseudoType)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnyToCty(tt.in)
			if err != nil {
				t.Fatalf("AnyToCty() error = %v", err)
			}
			if !got.Type().Equals(tt.want) {
				t.Errorf("AnyToCty(%#v) type = %s, want %s", tt.in, got.Type().GoString(), tt.want.GoString())
			}
		})
	}
}

// The point of the change: the same field set must produce the same type
// regardless of the values carried in it.
func TestAnyToCtyTypeIsIndependentOfValues(t *testing.T) {
	homogeneous := map[string]any{"a": 1, "b": 2}
	heterogeneous := map[string]any{"a": 1, "b": "x"}

	h1, err := AnyToCty(homogeneous)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := AnyToCty(heterogeneous)
	if err != nil {
		t.Fatal(err)
	}

	if h1.Type().IsMapType() || h2.Type().IsMapType() {
		t.Errorf("both must be objects; got %s and %s", h1.Type().GoString(), h2.Type().GoString())
	}
	// Both carry the same field names, so both must expose them as attributes.
	for _, v := range []cty.Value{h1, h2} {
		for _, attr := range []string{"a", "b"} {
			if !v.Type().HasAttribute(attr) {
				t.Errorf("%s missing attribute %q", v.Type().GoString(), attr)
			}
		}
	}
}

// Round-tripping a JSON-shaped record must still yield the original data.
func TestAnyToCtyRecordRoundTrips(t *testing.T) {
	in := map[string]any{"a": 1, "b": 2}
	cv, err := AnyToCty(in)
	if err != nil {
		t.Fatal(err)
	}
	back, err := CtyToAny(cv)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := back.(map[string]any)
	if !ok {
		t.Fatalf("CtyToAny() = %T, want map[string]any", back)
	}
	if m["a"] != 1 || m["b"] != 2 {
		t.Errorf("round-trip = %#v, want a=1 b=2", m)
	}
}
