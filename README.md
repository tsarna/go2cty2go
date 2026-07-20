# go2cty2go

Enhanced conversion utilities for converting between Go native types and HashiCorp's go-cty values.

## Overview

go2cty2go provides improved conversion functions between Go's native types and the cty value system used by HashiCorp Configuration Language (HCL). While the standard `gocty.ToCtyValue` and `gocty.FromCtyValue` functions handle basic conversions, this library offers several key advantages for complex data structures and edge cases.

## Installation

```bash
go get github.com/tsarna/go2cty2go
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/tsarna/go2cty2go"
    "github.com/zclconf/go-cty/cty"
)

func main() {
    // Convert Go value to cty.Value
    goValue := map[string]any{
        "name": "Alice",
        "scores": []int{85, 92, 78},
        "metadata": map[string]string{"role": "admin"},
    }
    
    ctyValue, err := go2cty2go.AnyToCty(goValue)
    if err != nil {
        panic(err)
    }
    
    // Convert cty.Value back to Go
    converted, err := go2cty2go.CtyToAny(ctyValue)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Converted: %+v\n", converted)
}
```

## Key Advantages over gocty

### 1. Recursive Collection Handling

**Standard gocty limitation:**
```go
// This fails with gocty.ToCtyValue
complexData := map[string]any{
    "users": []map[string]any{
        {"name": "Alice", "active": true},
        {"name": "Bob", "active": false},
    },
}
// Error: can't convert Go slice dynamically; only cty.Value allowed
```

**go2cty2go solution:**
```go
// This works seamlessly
ctyValue, err := go2cty2go.AnyToCty(complexData)
// Successfully creates proper cty object with nested structures
```

### 2. Intelligent Type Detection

go2cty2go automatically chooses the most appropriate cty type:

- **Homogeneous slices** → `cty.List`
- **Heterogeneous slices** → `cty.Tuple`
- **`map[string]any`** → `cty.Object` (always)
- **Maps with a concrete element type** (`map[string]string`, …) → `cty.Map`

Maps are typed from the map's **declared element type**, not from the values it
happens to hold. A `map[string]any` is a record — it is what JSON decoding
produces — so it becomes an object whether or not its values share a type.
Typing it from the values would mean the same field set produced a different
cty type depending on the data, which in turn changes which functions accept
it (`lookup` with a default of a different type, for example, is valid on an
object but not on a homogeneous map).

**Example:**
```go
// map[string]any is a record - always an object, regardless of values
record := map[string]any{"a": 1, "b": 2}
result, _ := go2cty2go.AnyToCty(record)
// result.Type() → cty.Object({a: number, b: number})

// A concretely-typed map stays a map
typed := map[string]string{"a": "x", "b": "y"}
result, _ = go2cty2go.AnyToCty(typed)
// result.Type() → cty.Map(string)

// Mixed types in slice - becomes cty.Tuple
mixed := []any{"hello", 42, true}
result, _ = go2cty2go.AnyToCty(mixed)
// result.Type() → cty.Tuple([string, number, bool])

// Same types in slice - becomes cty.List
uniform := []string{"a", "b", "c"}
result, _ = go2cty2go.AnyToCty(uniform)
// result.Type() → cty.List(string)
```

### 3. Enhanced Number Handling

Preserves integer precision when possible:

```go
// go2cty2go preserves exact integers
ctyNum := cty.NumberIntVal(42)
result, _ := go2cty2go.CtyToAny(ctyNum)
// result is int64(42), not float64(42)
```

### 4. Byte Slice Support

Handles `[]byte` gracefully by converting to strings:

```go
data := []byte("hello world")
ctyValue, _ := go2cty2go.AnyToCty(data)
// Creates cty.String("hello world") instead of failing
```

### 5. Capsule Type Unwrapping

Automatically unwraps cty capsule types:

```go
// Capsules are automatically unwrapped to their contained values
capsule := cty.CapsuleVal(someType, &someValue)
unwrapped, _ := go2cty2go.CtyToAny(capsule)
// unwrapped contains the actual Go value, not the capsule wrapper
```

### 6. Pointer and Interface Handling

Safely handles pointers and interface types:

```go
var ptr *string = &"hello"
var iface any = 42

// Both convert correctly without panics
ctyPtr, _ := go2cty2go.AnyToCty(ptr)     // → cty.String("hello")
ctyIface, _ := go2cty2go.AnyToCty(iface) // → cty.Number(42)
```

### 7. Struct Conversion via JSON

For complex structs, uses JSON marshaling as an intermediate step:

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

user := User{Name: "Alice", Email: "alice@example.com", Age: 30}
ctyValue, _ := go2cty2go.AnyToCty(user)
// Creates proper cty.Object with JSON field names
```

## Performance Characteristics

- **Direct type handling** for primitives (no reflection overhead)
- **Recursive processing** only when needed for collections
- **Fallback to gocty** for truly unknown types
- **Single-pass conversion** for most data structures

## Error Handling

All functions return descriptive errors with context:

```go
_, err := go2cty2go.AnyToCty(unsupportedType)
// Error: "failed to convert slice element 2: unsupported type chan int"
```

### Unknown values

`CtyToAny` reports an error for unknown values rather than converting them:

```go
_, err := go2cty2go.CtyToAny(cty.UnknownVal(cty.String))
// Error: "cannot convert unknown value of type string"
```

An unknown has no Go representation. It is deliberately not converted to
`nil` — that is reserved for nulls, and collapsing an unknown to `nil` would
let a not-yet-computed value be serialized as though the data were absent.
Unknowns nested inside a collection are reported by the recursive call, so
the error names the offending element.

`AnyToCty` passes a `cty.Value` through unchanged, unknown or not.

## Contributing

Contributions are welcome! Please ensure all tests pass and add tests for new functionality.

```bash
go test -v
```