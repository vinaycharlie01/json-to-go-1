package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var allOmitemptys bool
var flattens bool
var examples bool
var data interface{}
var scope interface{}
var goCode string
var tabs int
var seen = make(map[string]interface{})
var stack []string
var accumulator string
var innerTabs int
var parent string

func Format(str string) string {
	str = formatNumber(str)

	sanitized := ToProperCase(str)
	re := regexp.MustCompile("[^a-zA-Z0-9]")
	sanitized = re.ReplaceAllString(sanitized, "")

	if sanitized == "" {
		return "NAMING_FAILED"
	}

	// After sanitizing, the remaining characters can start with a number.
	// Run the sanitized string again through formatNumber to ensure the identifier is Num[0-9] or Zero_... instead of 1.
	return formatNumber(sanitized)
}

// formatNumber adds a prefix to a number to make an appropriate identifier in Go
func formatNumber(str string) string {
	if str == "" {
		return ""
	} else if matched, _ := regexp.MatchString(`^\d+$`, str); matched {
		str = "Num" + str
	} else if strings.IndexAny(string(str[0]), "0123456789") != -1 {
		numbers := map[string]string{
			"0": "Zero_", "1": "One_", "2": "Two_", "3": "Three_",
			"4": "Four_", "5": "Five_", "6": "Six_", "7": "Seven_",
			"8": "Eight_", "9": "Nine_",
		}
		str = numbers[string(str[0])] + str[1:]
	}

	return str
}

func ToProperCase(str string) string {
	// Ensure that the SCREAMING_SNAKE_CASE is converted to snake_case
	if match, _ := regexp.MatchString("^[_A-Z0-9]+$", str); match {
		str = strings.ToLower(str)
	}

	// List of common initialisms
	commonInitialisms := map[string]bool{
		"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true, "DNS": true,
		"EOF": true, "GUID": true, "HTML": true, "HTTP": true, "HTTPS": true, "ID": true,
		"IP": true, "JSON": true, "LHS": true, "QPS": true, "RAM": true, "RHS": true,
		"RPC": true, "SLA": true, "SMTP": true, "SQL": true, "SSH": true, "TCP": true,
		"TLS": true, "TTL": true, "UDP": true, "UI": true, "UID": true, "UUID": true,
		"URI": true, "URL": true, "UTF8": true, "VM": true, "XML": true, "XMPP": true,
		"XSRF": true, "XSS": true,
	}

	// Convert the string to Proper Case
	re := regexp.MustCompile(`(^|[^a-zA-Z])([a-z]+)`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		parts := re.FindStringSubmatch(match)
		sep, frag := parts[1], parts[2]

		if commonInitialisms[strings.ToUpper(frag)] {
			return sep + strings.ToUpper(frag)
		} else {
			return sep + strings.ToUpper(frag[0:1]) + strings.ToLower(frag[1:])
		}
	})

	re = regexp.MustCompile(`([A-Z])([a-z]+)`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		parts := re.FindStringSubmatch(match)
		sep, frag := parts[1], parts[2]

		if commonInitialisms[sep+strings.ToUpper(frag)] {
			return (sep + frag)[0:]
		} else {
			return sep + frag
		}
	})

	return str
}

func compareObjectKeys(itemAKeys, itemBKeys []string) bool {
	lengthA := len(itemAKeys)
	lengthB := len(itemBKeys)

	// nothing to compare, probably identical
	if lengthA == 0 && lengthB == 0 {
		return true
	}

	// duh
	if lengthA != lengthB {
		return false
	}

	// Sort the slices to ensure order doesn't matter
	sort.Strings(itemAKeys)
	sort.Strings(itemBKeys)

	// Compare each element
	for i, item := range itemAKeys {
		if item != itemBKeys[i] {
			return false
		}
	}
	return true
}

func FormatScopeKeys(keys []string) []string {
	for i := range keys {
		keys[i] = Format(keys[i])
	}
	return keys
}

func CompareObjectKeys(itemAKeys, itemBKeys interface{}) bool {
	valA := reflect.ValueOf(itemAKeys)
	valB := reflect.ValueOf(itemBKeys)

	lengthA := valA.Len()
	lengthB := valB.Len()

	// nothing to compare, probably identical
	if lengthA == 0 && lengthB == 0 {
		return true
	}

	// duh
	if lengthA != lengthB {
		return false
	}

	// Sort the slices to ensure order doesn't matter
	sort.Slice(itemAKeys, func(i, j int) bool {
		return fmt.Sprintf("%v", valA.Index(i).Interface()) < fmt.Sprintf("%v", valA.Index(j).Interface())
	})
	sort.Slice(itemBKeys, func(i, j int) bool {
		return fmt.Sprintf("%v", valB.Index(i).Interface()) < fmt.Sprintf("%v", valB.Index(j).Interface())
	})

	// Compare each element
	for i := 0; i < lengthA; i++ {
		if fmt.Sprintf("%v", valA.Index(i).Interface()) != fmt.Sprintf("%v", valB.Index(i).Interface()) {
			return false
		}
	}
	return true
}

func CompareObjects(objectA, objectB interface{}) bool {
	typeObject := reflect.TypeOf(map[string]interface{}{})

	return reflect.TypeOf(objectA) == typeObject &&
		reflect.TypeOf(objectB) == typeObject
}

func GetOriginalName(unique string) string {
	reLiteralUUID := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidLength := 36

	if len(unique) >= uuidLength {
		tail := unique[len(unique)-uuidLength:]
		if reLiteralUUID.MatchString(tail) {
			return unique[:len(unique)-(uuidLength+1)]
		}
	}
	return unique
}
func Uuidv4() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		panic(err)
	}

	// Set version (4) and variant bits (2)
	uuid[6] = (uuid[6] & 0x0F) | 0x40
	uuid[8] = (uuid[8] & 0x3F) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// Given two types, returns the more specific of the two
func mostSpecificPossibleGoType(typ1, typ2 string) string {
	if len(typ1) >= 5 && typ1[:5] == "float" &&
		len(typ2) >= 3 && typ2[:3] == "int" {
		return typ1
	} else if len(typ1) >= 3 && typ1[:3] == "int" &&
		len(typ2) >= 5 && typ2[:5] == "float" {
		return typ2
	} else {
		return "any"
	}
}

// Determines the most appropriate Go type
func goType(val interface{}) string {
	if val == nil {
		return "interface{}"
	}

	switch v := val.(type) {
	case string:
		if matched, _ := regexp.MatchString(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(\+\d\d:\d\d|Z)$`, v); matched {
			return "time.Time"
		}
		return "string"
	case int:
		return "int"
	case int64:
		return "int64"
	case float64:
		return "float64"
	case bool:
		return "bool"
	case []interface{}:
		return "[]interface{}"
	case map[string]interface{}:
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

// uniqueTypeName generates a unique name to avoid duplicate struct field names.
// This function appends a number at the end of the field name.
func uniqueTypeName(name string, seen []string) string {
	if !contains(seen, name) {
		return name
	}

	i := 0
	for {
		newName := name + strconv.Itoa(i)
		if !contains(seen, newName) {
			return newName
		}
		i++
	}
}

// contains checks if a string is present in a slice of strings.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func jsonToGo(jsonStr string, typename string, flatten bool, example bool, allOmitempty bool) (string, error) {
	flattens = flatten
	allOmitemptys = allOmitempty
	examples = example

	// Replace floats to stay as floats
	jsonStr = strings.Replace(jsonStr, ":\\s*\\[?\\s*-?\\d*\\.0", ":$1.1", -1)

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}
	scope = data

	typename = Format(typename)

	goCode += ("type " + typename + " ")
	// Append(`type ${typename} `)
	parseScope(data, 0)
	if flatten {
		return goCode + accumulator, nil
	} else {
		return goCode, nil
	}
}

func Append(str string) {
	goCode += str
}
func parseScope(scope interface{}, depth int) {
	switch v := scope.(type) {
	case nil:
		// Do nothing for nil
	case []interface{}:
		var sliceType string

		for i, item := range v {
			thisType := goType(item)
			if sliceType == "" {
				sliceType = thisType
			} else if sliceType != thisType {
				sliceType = mostSpecificPossibleGoType(thisType, sliceType)
				if sliceType == "any" {
					break
				}
			}
			if i == 0 && depth >= 2 {
				appender("[]")
			}
		}

		if sliceType == "struct" {
			allFields := make(map[string]struct {
				value interface{}
				count int
			})

			for _, item := range v {
				fields := reflect.ValueOf(item)
				for i := 0; i < fields.NumField(); i++ {
					keyname := fields.Type().Field(i).Name
					elem, ok := allFields[keyname]
					if !ok {
						elem = struct {
							value interface{}
							count int
						}{value: fields.Field(i).Interface()}
					} else {
						existingValue := elem.value
						currentValue := fields.Field(i).Interface()

						if CompareObjects(existingValue, currentValue) {
							comparisonResult := CompareObjectKeys(
								reflect.ValueOf(currentValue).MapKeys(),
								reflect.ValueOf(existingValue).MapKeys(),
							)
							if !comparisonResult {
								keyname = fmt.Sprintf("%s_%s", keyname, Uuidv4())
								elem = struct {
									value interface{}
									count int
								}{value: currentValue}
							}
						}
					}
					elem.count++
					allFields[keyname] = elem
				}
			}

			keys := reflect.ValueOf(allFields).MapKeys()
			structFields := make(map[string]interface{})
			omitempty := make(map[string]bool)

			for _, key := range keys {
				keyname := key.Interface().(string)
				elem := allFields[keyname]

				structFields[keyname] = elem.value
				omitempty[keyname] = elem.count != len(v)
			}

			parseStruct(depth+1, innerTabs, structFields, omitempty)
		} else if sliceType == "slice" {
			parseScope(v[0], depth)
		} else {
			if depth >= 2 {
				appender("[]" + sliceType)
			} else {
				Append("[]" + sliceType)
			}
		}
	case map[string]interface{}:
		if depth >= 2 {
			appender(parent)
		} else {
			Append(parent)
		}
		parseStruct(depth+1, innerTabs, v, nil)
	default:
		if depth >= 2 {
			appender(goType(v))
		} else {
			Append(goType(v))
		}
	}
}

func extractKeys(keys []reflect.Value) []string {
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = key.Interface().(string)
	}
	return result
}

func parseStruct(depth int, innerTabs int, scope map[string]interface{}, omitempty map[string]bool) {
	if flattens {
		stack = append(stack, strings.Repeat("\t", innerTabs))
	}

	// var seenTypeNames []string{}

	seenTypeNames := []string{}
	if flattens && depth >= 2 {
		parentType := fmt.Sprintf("type %s struct {\n", parent)
		scopeKeys := FormatScopeKeys(extractKeys(reflect.ValueOf(scope).MapKeys()))
		if seen, ok := seen[parent]; ok && reflect.DeepEqual(scopeKeys, seen) {
			stack = stack[:len(stack)-1]
			return
		}
		seen[parent] = scopeKeys

		appender(parentType)
		innerTabs++
		keys := reflect.ValueOf(scope).MapKeys()
		for _, key := range keys {
			keyname := GetOriginalName(key.String())
			indenter(innerTabs)
			typename := uniqueTypeName(Format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			appender(typename + " ")
			parent = typename
			parseScope(scope[key.String()], depth)
			appender(fmt.Sprintf("`json:\"%s", keyname))
			if allOmitemptys || (omitempty != nil && omitempty[key.String()]) {
				appender(",omitempty")
			}
			appender("\"\n")
		}
		indenter(innerTabs - 1)
		appender("}")
	} else {
		Append("struct {\n")
		tabs++
		keys := reflect.ValueOf(scope).MapKeys()
		for _, key := range keys {
			keyname := GetOriginalName(key.String())
			indent(tabs)
			typename := uniqueTypeName(Format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			Append(typename + " ")
			parent = typename
			parseScope(scope[key.String()], depth)
			Append(fmt.Sprintf("`json:\"%s", keyname))
			if allOmitemptys || (omitempty != nil && omitempty[key.String()]) {
				Append(",omitempty")
			}
			if examples && scope[key.String()] != "" && reflect.TypeOf(scope[key.String()]).Kind() != reflect.Map {
				Append(fmt.Sprintf("\" example:\"%v", scope[key.String()]))
			}
			Append("\"\n")
		}
		indent(tabs - 1)
		Append("}")
	}

	if flattens {
		accumulator += stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}
}

func appender(str string) {
	stack[len(stack)-1] += str
}

func indent(tabs int) {
	for i := 0; i < tabs; i++ {
		goCode += "\t"
	}
}

func indenter(tabs int) {
	for i := 0; i < tabs; i++ {
		stack[len(stack)-1] += "\t"
	}
}
func main() {

	jsonStr := `{"name": "John", "age": "2032/22/302", "city": "New York"}`
	typeName := "Person"

	// byte, _ := os.ReadFile("sample.json")

	// jsonStr := (string(byte))

	res, _ := jsonToGo(jsonStr, typeName, false, false, true)
	fmt.Println(res)

	// fmt.Println(goType(nil)) // Output: "interface{}"

	// fmt.Println(goType("2023-12-23T12:34:56Z")) // Output: "time.Time"
	// fmt.Println(goType("hello"))                // Output: "string"

	// fmt.Println(goType(42))                         // Output: "int"
	// fmt.Println(goType(int64(1234567890123456789))) // Output: "int64"
	// fmt.Println(goType(3.14))                       // Output: "float64"

	// fmt.Println(goType(true)) // Output: "bool"

	// fmt.Println(goType([]interface{}{1, 2, 3}))           // Output: "[]interface{}"
	// fmt.Println(goType(map[string]interface{}{"key": 1})) // Output: "map[string]interface{}"

	// typ1 := "float32"
	// typ2 := "int64"
	// result := mostSpecificPossibleGoType(typ1, typ2)
	// fmt.Println(result)

	// generatedUUID := Uuidv4()
	// fmt.Println(generatedUUID)

	// input1 := "exampleName-123e4567-e89b-12d3-a456-426614174000"
	// input2 := "anotherName"

	// result1 := GetOriginalName(input1)
	// result2 := GetOriginalName(input2)

	// fmt.Println(result1) // Should print "exampleName"
	// fmt.Println(result2) // Should print "anotherName"

	// inputs := []string{
	// 	"some identifier",
	// 	"another-identifier",
	// 	"123Identifier",
	// 	"!@#Special$%^Chars",
	// 	"CamelCaseIdentifier",
	// 	"1InvalidIdentifier",
	// }

	// for _, input := range inputs {
	// 	result := Format(input)
	// 	fmt.Printf("Input: %s => Output: %s\n", input, result)
	// }

	// 	// Example 1: Arrays with the same keys
	// 	keys1 := []string{"name", "age", "gender"}
	// 	keys2 := []string{"name", "age", "gender"}

	// 	fmt.Println(compareObjectKeys(keys1, keys2)) // Output: true

	// Example 2: Arrays with different order but same keys
	// keys3 := []interface{}{1, "gender", "name"}
	// keys4 := []interface{}{1, "age", "gender"}

	// fmt.Println(CompareObjectKeys(keys3, keys4)) // Output: true

	// 	// Example 3: Arrays with different keys
	// 	keys5 := []string{"name", "age", "gender"}
	// 	keys6 := []string{"name", "occupation", "gender"}

	// fmt.Println(compareObjectKeys(keys5, keys6)) // Output: false

	// Example 1: Both are map[string]interface{}
	// obj1 := map[string]interface{}{"name": "John", "age": 30}
	// obj2 := map[string]interface{}{"name": "Alice", "age": 25}

	// fmt.Println(CompareObjects(obj1, obj2)) // Output: true

	// // Example 2: One of them is not map[string]interface{}
	// obj3 := "Hello"
	// notAnObject := "Hello"

	// fmt.Println(CompareObjects(obj3, notAnObject)) // Output: false

	// // Example 3: Both are not map[string]interface{}
	// notAnObject1 := 42
	// notAnObject2 := "Hello, world!"

	// fmt.Println(CompareObjects(notAnObject1, notAnObject2))
}
