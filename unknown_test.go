package go2cty2go

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// callCtyToAny reports whether CtyToAny panicked, alongside its results.
// Every case here previously panicked instead of returning an error.
func callCtyToAny(v cty.Value) (out any, err error, panicked any) {
	defer func() { panicked = recover() }()
	out, err = CtyToAny(v)
	return
}

func TestCtyToAnyUnknownReturnsErrorNotPanic(t *testing.T) {
	tests := []struct {
		name string
		val  cty.Value
	}{
		{"string", cty.UnknownVal(cty.String)},
		{"number", cty.UnknownVal(cty.Number)},
		{"bool", cty.UnknownVal(cty.Bool)},
		{"list", cty.UnknownVal(cty.List(cty.String))},
		{"set", cty.UnknownVal(cty.Set(cty.String))},
		{"map", cty.UnknownVal(cty.Map(cty.String))},
		{"object", cty.UnknownVal(cty.Object(map[string]cty.Type{"a": cty.String}))},
		{"tuple", cty.UnknownVal(cty.Tuple([]cty.Type{cty.String}))},
		{"dynamic", cty.UnknownVal(cty.DynamicPseudoType)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err, panicked := callCtyToAny(tt.val)
			if panicked != nil {
				t.Fatalf("CtyToAny(unknown %s) panicked: %v", tt.name, panicked)
			}
			if err == nil {
				t.Fatalf("CtyToAny(unknown %s) = %#v, want an error", tt.name, out)
			}
			if !strings.Contains(err.Error(), "unknown") {
				t.Errorf("error = %q, want it to mention the value is unknown", err)
			}
		})
	}
}

func TestCtyToAnyUnknownNestedInCollection(t *testing.T) {
	// The collection itself is known; only an element is unknown, so the
	// top-level guard does not fire and the recursive call must catch it.
	tests := []struct {
		name string
		val  cty.Value
	}{
		{"list element", cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)})},
		{"list element after known", cty.ListVal([]cty.Value{cty.StringVal("ok"), cty.UnknownVal(cty.String)})},
		{"set element", cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)})},
		{"map value", cty.MapVal(map[string]cty.Value{"k": cty.UnknownVal(cty.String)})},
		{"object attribute", cty.ObjectVal(map[string]cty.Value{"a": cty.UnknownVal(cty.String)})},
		{"tuple element", cty.TupleVal([]cty.Value{cty.UnknownVal(cty.Bool)})},
		{"deeply nested", cty.ObjectVal(map[string]cty.Value{
			"outer": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"inner": cty.UnknownVal(cty.Number)}),
			}),
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err, panicked := callCtyToAny(tt.val)
			if panicked != nil {
				t.Fatalf("CtyToAny(%s) panicked: %v", tt.name, panicked)
			}
			if err == nil {
				t.Fatalf("CtyToAny(%s) = %#v, want an error", tt.name, out)
			}
		})
	}
}

func TestCtyToAnyNullStillConvertsToNil(t *testing.T) {
	// A null is known and must keep its existing behavior — the unknown
	// guard must not capture it.
	for _, ty := range []cty.Type{cty.String, cty.Number, cty.Bool, cty.DynamicPseudoType} {
		out, err := CtyToAny(cty.NullVal(ty))
		if err != nil {
			t.Errorf("CtyToAny(null %s) error = %v, want nil", ty.FriendlyName(), err)
		}
		if out != nil {
			t.Errorf("CtyToAny(null %s) = %#v, want nil", ty.FriendlyName(), out)
		}
	}
}

func TestCtyToAnyKnownValuesUnaffected(t *testing.T) {
	// Guard against the unknown check accidentally rejecting known values.
	cases := map[string]cty.Value{
		"string": cty.StringVal("s"),
		"number": cty.NumberIntVal(1),
		"bool":   cty.True,
		"list":   cty.ListVal([]cty.Value{cty.StringVal("a")}),
		"map":    cty.MapVal(map[string]cty.Value{"k": cty.StringVal("v")}),
		"object": cty.ObjectVal(map[string]cty.Value{"a": cty.StringVal("b")}),
	}
	for name, v := range cases {
		if _, err := CtyToAny(v); err != nil {
			t.Errorf("CtyToAny(known %s) error = %v, want nil", name, err)
		}
	}
}
