package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

var data interface{}
var scope interface{}
var goCode strings.Builder
var tabs int
var seen = make(map[string]bool)
var stack []string
var accumulator strings.Builder
var innerTabs int
var parent string
var flatten bool
var allOmitempty bool

// jsonToGo converts JSON to Go code
func jsonToGo(jsonStr string, typename string, flatten bool, example bool, allOmitempty bool) {
	// Assign the function parameters to package-level variables
	flatten = flatten
	allOmitempty = allOmitempty

	// Replace floats to stay as floats
	jsonStr = strings.Replace(jsonStr, ":\\s*\\[?\\s*-?\\d*\\.0", ":$1.1", -1)

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		log.Fatal(err)
	}
	scope = data

	// Clear global variables
	goCode.Reset()
	stack = nil
	accumulator.Reset()
	seen = make(map[string]bool)

	// Format typename
	// typename = format(typename)

	goCode.WriteString("type " + typename + " ")

}

// Other functions remain unchanged...

func main() {
	jsonStr := `{"name": "John", "age": 30, "city": "New York"}`
	typeName := "Person"

	jsonToGo(jsonStr, typeName, false, false, false)

	fmt.Println(goCode.String())
}
