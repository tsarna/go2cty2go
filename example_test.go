package go2cty2go_test

import (
	"fmt"
	"log"

	"github.com/tsarna/go2cty2go"
	"github.com/zclconf/go-cty/cty"
)

func ExampleAnyToCty() {
	// Complex nested data structure
	data := map[string]any{
		"users": []map[string]any{
			{"name": "Alice", "age": 30, "active": true},
			{"name": "Bob", "age": 25, "active": false},
		},
		"settings": map[string]any{
			"theme":    "dark",
			"language": "en",
			"features": []string{"notifications", "sync"},
		},
		"metadata": map[string]any{
			"version":   "1.2.3",
			"buildTime": "2023-01-01T00:00:00Z",
		},
	}

	// Convert to cty.Value
	ctyValue, err := go2cty2go.AnyToCty(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted to cty type: %s\n", ctyValue.Type().FriendlyName())

	// Access nested values
	users := ctyValue.GetAttr("users")
	fmt.Printf("Users type: %s\n", users.Type().FriendlyName())

	// Output:
	// Converted to cty type: object
	// Users type: list of object
}

func ExampleCtyToAny() {
	// Create a cty.Value
	ctyValue := cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("Alice"),
		"age":  cty.NumberIntVal(30),
		"scores": cty.ListVal([]cty.Value{
			cty.NumberIntVal(85),
			cty.NumberIntVal(92),
			cty.NumberIntVal(78),
		}),
		"metadata": cty.ObjectVal(map[string]cty.Value{
			"role": cty.StringVal("admin"),
			"team": cty.StringVal("engineering"),
		}),
	})

	// Convert to Go native types
	result, err := go2cty2go.CtyToAny(ctyValue)
	if err != nil {
		log.Fatal(err)
	}

	// Type assertion to access the data
	data := result.(map[string]any)
	fmt.Printf("Name: %s\n", data["name"])
	fmt.Printf("Age: %d\n", data["age"])
	fmt.Printf("Scores: %v\n", data["scores"])

	// Output:
	// Name: Alice
	// Age: 30
	// Scores: [85 92 78]
}

func ExampleAnyToCty_byteSlice() {
	// Byte slices are converted to strings
	data := []byte("Hello, World!")

	ctyValue, err := go2cty2go.AnyToCty(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type: %s\n", ctyValue.Type().FriendlyName())
	fmt.Printf("Value: %s\n", ctyValue.AsString())

	// Output:
	// Type: string
	// Value: Hello, World!
}

func ExampleAnyToCty_mixedTypes() {
	// Mixed-type slice becomes a tuple
	mixedSlice := []any{"hello", 42, true, 3.14}

	ctyValue, err := go2cty2go.AnyToCty(mixedSlice)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type: %s\n", ctyValue.Type().FriendlyName())

	// Uniform-type slice becomes a list
	uniformSlice := []string{"apple", "banana", "cherry"}

	ctyValue2, err := go2cty2go.AnyToCty(uniformSlice)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type: %s\n", ctyValue2.Type().FriendlyName())

	// Output:
	// Type: tuple
	// Type: list of string
}
