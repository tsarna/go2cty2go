package go2cty2go

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// --- fixtures -------------------------------------------------------------

// ctyMarshalerType is a Go value that dictates its own cty form.
type ctyMarshalerType struct{ n int }

func (m ctyMarshalerType) ToCty() (cty.Value, error) {
	return cty.StringVal("marshaled:" + string(rune('0'+m.n))), nil
}

// erroringMarshaler reports an error from ToCty.
type erroringMarshaler struct{}

func (erroringMarshaler) ToCty() (cty.Value, error) {
	return cty.NilVal, errors.New("marshal boom")
}

// nativeThing is stored inside a capsule and knows its native form. It also
// records the CapsuleInfo it was handed so tests can assert the container.
type nativeThing struct {
	payload   string
	gotInfo   *CapsuleInfo
	returnErr error
}

func (n *nativeThing) CtyToNativeValue(info CapsuleInfo) (any, error) {
	n.gotInfo = &info
	if n.returnErr != nil {
		return nil, n.returnErr
	}
	return n.payload, nil
}

// plainCapsuleThing is stored inside a capsule but does NOT implement
// NativeMarshaler, so it must fall back to being returned unwrapped.
type plainCapsuleThing struct{ v int }

var (
	nativeCapsuleType = cty.CapsuleWithOps("nativeThing", reflect.TypeOf(nativeThing{}), &cty.CapsuleOps{})
	plainCapsuleType  = cty.Capsule("plainThing", reflect.TypeOf(plainCapsuleThing{}))
)

func nativeCapsule(n *nativeThing) cty.Value { return cty.CapsuleVal(nativeCapsuleType, n) }

// --- AnyToCty: ToCty -----------------------------------------------------

func TestAnyToCty_UsesCtyMarshaler(t *testing.T) {
	got, err := AnyToCty(ctyMarshalerType{n: 7})
	if err != nil {
		t.Fatalf("AnyToCty() error = %v", err)
	}
	if got != cty.StringVal("marshaled:7") {
		t.Errorf("AnyToCty() = %#v, want the ToCty result", got)
	}
}

func TestAnyToCty_CtyMarshalerViaPointer(t *testing.T) {
	// Method set is on the value type, so a pointer satisfies it too.
	got, err := AnyToCty(&ctyMarshalerType{n: 3})
	if err != nil {
		t.Fatalf("AnyToCty() error = %v", err)
	}
	if got != cty.StringVal("marshaled:3") {
		t.Errorf("AnyToCty(ptr) = %#v, want the ToCty result", got)
	}
}

func TestAnyToCty_CtyMarshalerErrorPropagates(t *testing.T) {
	_, err := AnyToCty(erroringMarshaler{})
	if err == nil || err.Error() != "marshal boom" {
		t.Fatalf("AnyToCty() error = %v, want \"marshal boom\"", err)
	}
}

// pointerMarshaler implements ToCty only on the pointer receiver, so a nil
// pointer of this type is a CtyMarshaler whose method would deref nil.
type pointerMarshaler struct{ s string }

func (p *pointerMarshaler) ToCty() (cty.Value, error) { return cty.StringVal(p.s), nil }

func TestAnyToCty_NilPointerMarshalerIsNullNotPanic(t *testing.T) {
	var p *pointerMarshaler // typed nil, implements CtyMarshaler
	got, err := AnyToCty(p)
	if err != nil {
		t.Fatalf("AnyToCty(nil ptr) error = %v", err)
	}
	if !got.IsNull() {
		t.Errorf("AnyToCty(nil ptr marshaler) = %#v, want null", got)
	}
}

// --- CtyToAny: bare capsule ----------------------------------------------

func TestCtyToAny_BareCapsuleUsesNativeMarshaler(t *testing.T) {
	n := &nativeThing{payload: "hello"}
	got, err := CtyToAny(nativeCapsule(n))
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	if got != "hello" {
		t.Errorf("CtyToAny(capsule) = %#v, want \"hello\"", got)
	}
	if n.gotInfo == nil {
		t.Fatal("CtyToNativeValue was not called")
	}
	if n.gotInfo.Container != cty.NilVal {
		t.Errorf("bare capsule Container = %#v, want cty.NilVal", n.gotInfo.Container)
	}
}

func TestCtyToAny_CapsuleWithoutNativeMarshalerUnwrapsAsBefore(t *testing.T) {
	// Backwards-compatible: a capsule whose value does not implement the
	// interface still returns the raw encapsulated value.
	orig := &plainCapsuleThing{v: 42}
	got, err := CtyToAny(cty.CapsuleVal(plainCapsuleType, orig))
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	if got != orig {
		t.Errorf("CtyToAny(plain capsule) = %#v, want the encapsulated pointer", got)
	}
}

func TestCtyToAny_NativeMarshalerErrorPropagates(t *testing.T) {
	n := &nativeThing{returnErr: errors.New("native boom")}
	_, err := CtyToAny(nativeCapsule(n))
	if err == nil || err.Error() != "native boom" {
		t.Fatalf("CtyToAny() error = %v, want \"native boom\"", err)
	}
}

// --- CtyToAny: _capsule rich object --------------------------------------

func TestCtyToAny_RichObjectUsesNativeMarshalerWithContainer(t *testing.T) {
	n := &nativeThing{payload: "rich"}
	obj := cty.ObjectVal(map[string]cty.Value{
		"content_type": cty.StringVal("text/plain"),
		"_capsule":     nativeCapsule(n),
	})

	got, err := CtyToAny(obj)
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	if got != "rich" {
		t.Errorf("CtyToAny(rich object) = %#v, want \"rich\"", got)
	}
	if n.gotInfo == nil {
		t.Fatal("CtyToNativeValue was not called")
	}
	// The whole object must be handed through so the value can read siblings.
	if !n.gotInfo.Container.Type().IsObjectType() {
		t.Fatalf("Container = %#v, want the enclosing object", n.gotInfo.Container)
	}
	if ct := n.gotInfo.Container.GetAttr("content_type").AsString(); ct != "text/plain" {
		t.Errorf("Container.content_type = %q, want \"text/plain\"", ct)
	}
}

func TestCtyToAny_ObjectWithoutCapsuleIsPlainRecord(t *testing.T) {
	// An ordinary object — no _capsule — converts attribute by attribute.
	obj := cty.ObjectVal(map[string]cty.Value{
		"a": cty.StringVal("x"),
		"b": cty.NumberIntVal(2),
	})
	got, err := CtyToAny(obj)
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("CtyToAny(object) = %T, want map[string]any", got)
	}
	if m["a"] != "x" || m["b"] != 2 {
		t.Errorf("CtyToAny(object) = %#v, want a=x b=2", m)
	}
}

func TestCtyToAny_CapsuleAttributeNotACapsuleIsPlainRecord(t *testing.T) {
	// A stray attribute literally named _capsule that is not a capsule must
	// not hijack conversion; the object stays a plain record.
	obj := cty.ObjectVal(map[string]cty.Value{
		"_capsule": cty.StringVal("not really a capsule"),
	})
	got, err := CtyToAny(obj)
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("CtyToAny() = %T, want map[string]any", got)
	}
	if m["_capsule"] != "not really a capsule" {
		t.Errorf("CtyToAny() = %#v, want the string preserved", m)
	}
}

// --- round trip ----------------------------------------------------------

// roundTripThing implements both directions: its native form is a string,
// and it rebuilds a capsule from that string.
type roundTripThing struct{ s string }

func (r roundTripThing) ToCty() (cty.Value, error) {
	return cty.CapsuleVal(roundTripType, &roundTripThing{s: r.s}), nil
}
func (r *roundTripThing) CtyToNativeValue(CapsuleInfo) (any, error) { return r.s, nil }

var roundTripType = cty.CapsuleWithOps("roundTrip", reflect.TypeOf(roundTripThing{}), &cty.CapsuleOps{})

func TestRoundTrip_ToCtyThenCtyToNative(t *testing.T) {
	cv, err := AnyToCty(roundTripThing{s: "payload"})
	if err != nil {
		t.Fatalf("AnyToCty() error = %v", err)
	}
	if !cv.Type().IsCapsuleType() {
		t.Fatalf("AnyToCty() type = %s, want a capsule", cv.Type().GoString())
	}
	back, err := CtyToAny(cv)
	if err != nil {
		t.Fatalf("CtyToAny() error = %v", err)
	}
	if back != "payload" {
		t.Errorf("round trip = %#v, want \"payload\"", back)
	}
}
