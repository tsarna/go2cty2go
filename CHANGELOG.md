# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-07-20

### Added

- **Conversion interfaces for capsule and custom types.** A Go value can now control
  its own conversion in either direction, taking precedence over the built-in reflection
  and struct-JSON handling:

  - `CtyMarshaler` — `ToCty() (cty.Value, error)`. `AnyToCty` calls it in preference to
    reflecting over the value, so a type decides exactly what cty value it becomes. The
    `cty.Value` return keeps the method unambiguous, so it stays a plain structural
    interface — implementers need the method, not an import of this package.

  - `NativeMarshaler` — `CtyToNativeValue(CapsuleInfo) (any, error)`. When `CtyToAny`
    reaches a capsule whose encapsulated value implements this, it returns the method's
    result instead of the raw encapsulated value. The parameter is the owned `CapsuleInfo`
    struct (rather than a bare `cty.Value`) so the signature cannot be satisfied by
    accident and can gain fields later without breaking implementers; implementing this
    direction therefore requires importing this package.

- **Rich-object (`_capsule`) support in `CtyToAny`.** An object carrying a capsule under a
  `_capsule` attribute, whose wrapped value implements `NativeMarshaler`, converts via that
  value — with the enclosing object passed in `CapsuleInfo.Container` so it can read
  sibling attributes. Objects without such a `_capsule` are unchanged (converted attribute
  by attribute).

## [0.2.0] - 2026-07-20

### Changed

- **BREAKING: `map[string]any` is now always converted to a `cty.Object`.** Maps are typed
  from the map's declared element type, not from the values they happen to hold. Previously
  the converter inspected the runtime values and returned a `cty.Map` when they shared a
  type, a `cty.Object` otherwise — so the cty type of a decoded JSON record depended on its
  data (`{"a":1,"b":2}` → map, `{"a":1,"b":"x"}` → object). A `map[string]any` is a record,
  so it is now always an object; a map with a concrete element type (`map[string]string`, …)
  stays a `cty.Map`. Migration: attribute and index access in HCL are unaffected;
  `cty.Value.Index` panics on an object (use `GetAttr`), and `keys()` returns a tuple.

## [0.1.4] - 2026-07-20

### Fixed

- **`CtyToAny` returns an error for unknown values instead of panicking.** There was no
  `IsKnown` check, so every unknown value panicked from inside an accessor (`AsString`,
  `AsBigFloat`, `True`, `ElementIterator`), including an unknown nested inside an
  otherwise-known collection. Unknowns are reported as an error rather than converted to
  `nil` (which is reserved for nulls).

## [0.1.3] - 2026-04-18

### Changed

- Updated `go-cty` to v1.18.1.

## [0.1.2] - 2026-03-21

### Changed

- **`CtyToAny` returns `int` rather than `int64` for whole numbers**, for compatibility
  with JSON-based tools like `gojq`.

## [0.1.1] - 2026-03-21

### Changed

- Updated `go-cty` to v1.18.0.

### Added

- GitHub Actions CI workflow and Renovate configuration.

## [0.1.0] - 2025-09-18

### Added

- Initial release. Recursive conversion between Go native values and `cty.Value` in both
  directions (`AnyToCty` / `CtyToAny`), with intelligent type detection for slices and
  maps, capsule unwrapping, pointer/interface handling, and struct conversion via JSON.
