package go2cty2go

import "github.com/zclconf/go-cty/cty"

// CapsuleInfo carries context for a capsule value's native conversion. It is
// a struct rather than a bare parameter so that new context can be added as
// fields without breaking implementers of NativeMarshaler.
type CapsuleInfo struct {
	// Container is the enclosing object when the capsule was reached via the
	// _capsule rich-object convention, or cty.NilVal when CtyToAny was called
	// on a bare capsule. It lets a wrapped value read sibling attributes of
	// its rich object if the capsule alone is not self-describing.
	Container cty.Value
}

// CtyMarshaler is implemented by a Go value that knows its own cty
// representation. AnyToCty calls ToCty in preference to its built-in
// reflection and struct-JSON handling, so a type can decide exactly what it
// becomes.
//
// The interface is structural: an implementer needs the method, not an
// import of this package. The cty.Value return keeps the method unambiguous —
// a method of this shape can only mean "convert me to cty".
type CtyMarshaler interface {
	ToCty() (cty.Value, error)
}

// NativeMarshaler is implemented by the Go value inside a cty capsule to
// control its native representation. When CtyToAny reaches a capsule — either
// bare, or under the _capsule attribute of a rich object — and its
// encapsulated value implements this interface, CtyToAny returns the result
// of CtyToNativeValue instead of the raw encapsulated value.
//
// The method takes a CapsuleInfo (an owned type) rather than a bare cty.Value
// so the signature cannot be satisfied by accident: a foreign method would
// have to name go2cty2go.CapsuleInfo deliberately. Implementing this
// direction therefore requires importing this package; the ToCty direction
// does not.
type NativeMarshaler interface {
	CtyToNativeValue(CapsuleInfo) (any, error)
}
