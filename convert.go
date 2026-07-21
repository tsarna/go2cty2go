package go2cty2go

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// CtyToAny converts a cty.Value to Go native types recursively.
//
// Unknown values have no Go representation and are reported as an error
// rather than converted. Nil is deliberately not used for them: an unknown
// means "not yet computed", and collapsing it to nil would let a
// half-evaluated value be serialized as though the data were absent.
// Unknowns nested inside a collection are reported by the recursive call,
// so the error names the offending element.
func CtyToAny(message cty.Value) (any, error) {
	switch {
	case message.IsNull():
		return nil, nil
	case !message.IsKnown():
		// Checked before any accessor: AsString, True, AsBigFloat, and
		// ElementIterator all panic on an unknown value.
		return nil, fmt.Errorf("cannot convert unknown value of type %s", message.Type().FriendlyName())
	case message.Type() == cty.String:
		return message.AsString(), nil
	case message.Type() == cty.Bool:
		return message.True(), nil
	case message.Type() == cty.Number:
		// Try int first, then fall back to float.
		// Return int (not int64) for compatibility with JSON-based tools like gojq.
		if val, accuracy := message.AsBigFloat().Int64(); accuracy == big.Exact {
			return int(val), nil
		} else {
			val, _ := message.AsBigFloat().Float64()
			return val, nil
		}
	case message.Type().IsObjectType():
		// Rich-object convention: an object carrying a capsule under the
		// _capsule attribute, whose wrapped value knows its native form,
		// converts via that — passing the whole object so the value can read
		// its sibling attributes. Any object without such a _capsule is a
		// plain record and is converted attribute by attribute as before.
		if message.Type().HasAttribute("_capsule") {
			capAttr := message.GetAttr("_capsule")
			if capAttr.IsKnown() && !capAttr.IsNull() && capAttr.Type().IsCapsuleType() {
				if nm, ok := capAttr.EncapsulatedValue().(NativeMarshaler); ok {
					return nm.CtyToNativeValue(CapsuleInfo{Container: message})
				}
			}
		}
		result := make(map[string]any)
		for name := range message.Type().AttributeTypes() {
			attrVal := message.GetAttr(name)
			converted, err := CtyToAny(attrVal) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert attribute %s: %w", name, err)
			}
			result[name] = converted
		}
		return result, nil
	case message.Type().IsListType() || message.Type().IsTupleType():
		var result []any
		it := message.ElementIterator()
		for it.Next() {
			_, elemVal := it.Element()
			converted, err := CtyToAny(elemVal) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert array element: %w", err)
			}
			result = append(result, converted)
		}
		return result, nil
	case message.Type().IsSetType():
		var result []any
		it := message.ElementIterator()
		for it.Next() {
			_, elemVal := it.Element()
			converted, err := CtyToAny(elemVal) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert set element: %w", err)
			}
			result = append(result, converted)
		}
		return result, nil
	case message.Type().IsMapType():
		result := make(map[string]any)
		it := message.ElementIterator()
		for it.Next() {
			keyVal, elemVal := it.Element()
			key := keyVal.AsString()
			converted, err := CtyToAny(elemVal) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %s: %w", key, err)
			}
			result[key] = converted
		}
		return result, nil
	case message.Type().IsCapsuleType():
		// A capsule whose wrapped value knows its native form converts via
		// that; otherwise unwrap and return the encapsulated value as before.
		enc := message.EncapsulatedValue()
		if nm, ok := enc.(NativeMarshaler); ok {
			return nm.CtyToNativeValue(CapsuleInfo{Container: cty.NilVal})
		}
		return enc, nil
	default:
		// Fall back to gocty for truly unknown types
		var result any
		err := gocty.FromCtyValue(message, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to convert cty value of type %s: %w", message.Type().FriendlyName(), err)
		}
		return result, nil
	}
}

// AnyToCty converts a Go value to a cty.Value with recursive handling
func AnyToCty(v any) (cty.Value, error) {
	// Handle nil
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}

	// If it's already a cty.Value, return it as-is
	if ctyVal, ok := v.(cty.Value); ok {
		return ctyVal, nil
	}

	// A type that knows its own cty form takes precedence over the built-in
	// reflection and struct-JSON handling. Skip a typed-nil pointer so the
	// method is not invoked on a nil receiver (it falls through to the
	// pointer handling below, which yields a null).
	if m, ok := v.(CtyMarshaler); ok {
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr && rv.IsNil() {
			return cty.NullVal(cty.DynamicPseudoType), nil
		}
		return m.ToCty()
	}

	// Special case: []byte -> string conversion
	if bytes, ok := v.([]byte); ok {
		return cty.StringVal(string(bytes)), nil
	}

	// Handle primitive types directly
	switch val := v.(type) {
	case string:
		return cty.StringVal(val), nil
	case bool:
		return cty.BoolVal(val), nil
	case int:
		return cty.NumberIntVal(int64(val)), nil
	case int8:
		return cty.NumberIntVal(int64(val)), nil
	case int16:
		return cty.NumberIntVal(int64(val)), nil
	case int32:
		return cty.NumberIntVal(int64(val)), nil
	case int64:
		return cty.NumberIntVal(val), nil
	case uint:
		return cty.NumberUIntVal(uint64(val)), nil
	case uint8:
		return cty.NumberUIntVal(uint64(val)), nil
	case uint16:
		return cty.NumberUIntVal(uint64(val)), nil
	case uint32:
		return cty.NumberUIntVal(uint64(val)), nil
	case uint64:
		return cty.NumberUIntVal(val), nil
	case float32:
		return cty.NumberFloatVal(float64(val)), nil
	case float64:
		return cty.NumberFloatVal(val), nil
	}

	// Handle collections recursively using reflection
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)

	switch rt.Kind() {
	case reflect.Slice, reflect.Array:
		return convertSliceRecursively(rv)
	case reflect.Map:
		return convertMapRecursively(rv)
	case reflect.Ptr:
		// Dereference pointer and recurse
		if rv.IsNil() {
			return cty.NullVal(cty.DynamicPseudoType), nil
		}
		return AnyToCty(rv.Elem().Interface())
	case reflect.Interface:
		// Handle any by getting the concrete value
		if rv.IsNil() {
			return cty.NullVal(cty.DynamicPseudoType), nil
		}
		return AnyToCty(rv.Elem().Interface())
	case reflect.Struct:
		// For structs, try JSON conversion as they may have complex field mappings
		return tryJSONConversion(v)
	}

	// Last resort: try gocty direct conversion
	return gocty.ToCtyValue(v, cty.DynamicPseudoType)
}

// convertSliceRecursively converts slices and arrays to cty lists
func convertSliceRecursively(rv reflect.Value) (cty.Value, error) {
	length := rv.Len()
	if length == 0 {
		// Empty slice - create an empty list of dynamic type
		return cty.ListValEmpty(cty.DynamicPseudoType), nil
	}

	// Convert all elements recursively
	ctyValues := make([]cty.Value, length)
	for i := 0; i < length; i++ {
		elem := rv.Index(i).Interface()
		ctyElem, err := AnyToCty(elem)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to convert slice element %d: %w", i, err)
		}
		ctyValues[i] = ctyElem
	}

	// Try to create a list - if types are inconsistent, create a tuple
	if allSameType(ctyValues) {
		return cty.ListVal(ctyValues), nil
	} else {
		return cty.TupleVal(ctyValues), nil
	}
}

// convertMapRecursively converts maps to cty maps or objects
func convertMapRecursively(rv reflect.Value) (cty.Value, error) {
	// Only support string keys for now (cty limitation)
	if rv.Type().Key().Kind() != reflect.String {
		return cty.NilVal, fmt.Errorf("map keys must be strings, got %s", rv.Type().Key().Kind())
	}

	// A map[string]any is a record whose fields happen to be carried in a
	// map — that is what JSON decoding produces. Its cty type must not
	// depend on whether the current values happen to share a type, so it
	// always becomes an object. A map with a concrete element type is a
	// genuine homogeneous map and stays one.
	asObject := rv.Type().Elem().Kind() == reflect.Interface

	if rv.Len() == 0 {
		if asObject {
			return cty.EmptyObjectVal, nil
		}
		// Empty map - create an empty map of dynamic type
		return cty.MapValEmpty(cty.DynamicPseudoType), nil
	}

	// Convert all key-value pairs recursively
	ctyMap := make(map[string]cty.Value)
	for _, key := range rv.MapKeys() {
		keyStr := key.String()
		value := rv.MapIndex(key).Interface()

		ctyValue, err := AnyToCty(value)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to convert map value for key %q: %w", keyStr, err)
		}
		ctyMap[keyStr] = ctyValue
	}

	// Check if all values have the same type - if so, create a map, otherwise an object
	values := make([]cty.Value, 0, len(ctyMap))
	for _, v := range ctyMap {
		values = append(values, v)
	}

	if !asObject && allSameType(values) {
		return cty.MapVal(ctyMap), nil
	}
	// Mixed types, or a map[string]any — use an object.
	return cty.ObjectVal(ctyMap), nil
}

// allSameType checks if all cty values have the same type
func allSameType(values []cty.Value) bool {
	if len(values) <= 1 {
		return true
	}

	firstType := values[0].Type()
	for i := 1; i < len(values); i++ {
		if !values[i].Type().Equals(firstType) {
			return false
		}
	}
	return true
}

// tryJSONConversion attempts conversion via JSON marshaling then our own recursive conversion
func tryJSONConversion(v any) (cty.Value, error) {
	// Marshal to JSON to get a generic representation
	raw, err := json.Marshal(v)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to JSON marshal: %w", err)
	}

	// Unmarshal to a generic Go structure (map[string]any, []any, primitives)
	var genericValue any
	err = json.Unmarshal(raw, &genericValue)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return AnyToCty(genericValue)
}
