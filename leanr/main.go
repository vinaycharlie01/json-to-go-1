package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// let data;
// 	let scope;
// 	let go = "";
// 	let tabs = 0;

// 	const seen = {};
// 	const stack = [];
// 	let accumulator = "";
// 	let innerTabs = 0;
// 	let parent = "";

var allOmitemptys bool
var flattens bool
var data interface{}
var scope interface{}
var goCode string
var tabs int
var seen = make(map[string]bool)
var stack []string
var accumulator string
var innerTabs int
var parent string

func jsonToGo(jsonStr string, typename string, flatten bool, example bool, allOmitempty bool) (string, error) {
	flattens = flatten
	allOmitemptys = allOmitempty

	// Replace floats to stay as floats
	jsonStr = strings.Replace(jsonStr, ":\\s*\\[?\\s*-?\\d*\\.0", ":$1.1", -1)

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}
	scope = data

	typename = format(typename)

	goCode += ("type " + typename + " ")
	parseScope(data, 0, flatten, example)
	if flatten {
		return goCode + accumulator, nil
	} else {
		return goCode, nil
	}
}

func parseScope(scope interface{}, depth int, flatten, example bool) {
	switch s := scope.(type) {
	case map[string]interface{}:
		parseStruct(depth+1, innerTabs, s, nil, flatten, example)
	case []interface{}:
		parseSlice(depth+1, innerTabs, s, flatten)
	default:
		if flatten && depth >= 2 {
			appender(goType(scope))
		} else {
			Append(goType(scope))
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

func parseStruct(depth, innerTabs int, scope map[string]interface{}, omitempty map[string]bool, flatten, example bool) {
	if flatten {
		stack = append(stack, strings.Repeat("\t", depth-2)+"\n")
	}

	seenTypeNames := []string{}

	if flatten && depth >= 2 {
		parentType := fmt.Sprintf("type %s", parent)

		scopeKeys := formatScopeKeys(extractKeys(reflect.ValueOf(scope).MapKeys()))
		seenKeys, ok := seen[parent]
		if ok && compareObjectKeys(scopeKeys, seenKeys) {
			stack = stack[:len(stack)-1]
			return
		}
		seen[parent] = ok

		appender(fmt.Sprintf("%s struct {\n", parentType))
		innerTabs++
		keys := reflect.ValueOf(scope).MapKeys()
		for _, key := range keys {
			keyname := getOriginalName(key.String())
			indenter(innerTabs)
			typename := uniqueTypeName(format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			appender(typename + " ")
			parent = typename
			parseScope(scope[key.Interface().(string)], depth, flatten, example)
			appender(fmt.Sprintf(" `json:\"%s", keyname))
			if allOmitemptys || (omitempty != nil && omitempty[keyname]) {
				appender(",omitempty")
			}
			appender("\"`\n")
		}
		indenter(innerTabs - 1)
		appender("}")
	} else {
		appender("struct {\n")
		tabs++
		keys := reflect.ValueOf(scope).MapKeys()
		for _, key := range keys {
			keyname := getOriginalName(key.String())
			indenter(tabs)
			typename := uniqueTypeName(format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			appender(typename + " ")
			parent = typename
			parseScope(scope[key.Interface().(string)], depth, flatten, example)
			appender(fmt.Sprintf(" `json:\"%s", keyname))
			if allOmitemptys || (omitempty != nil && omitempty[keyname]) {
				appender(",omitempty")
			}
			_, ok := scope[key.Interface().(string)]
			if example && ok && reflect.TypeOf(scope[key.Interface().(string)]).Kind() != reflect.Map {
				appender(fmt.Sprintf("\" example:\"%v", scope[key.Interface().(string)]))
			}
			appender("\"\n")
		}
		indenter(tabs - 1)
		appender("}")
	}

	if flatten {
		accumulator += stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}
}

func parseSlice(depth, innerTabs int, scope []interface{}, example bool) {
	sliceType := ""
	scopeLength := len(scope)

	for i := 0; i < scopeLength; i++ {
		thisType := goType(scope[i])
		if sliceType == "" {
			sliceType = thisType
		} else if sliceType != thisType {
			sliceType = mostSpecificPossibleGoType(thisType, sliceType)
			if sliceType == "any" {
				break
			}
		}
	}

	slice := ""
	if flattens && depth >= 2 {
		slice = fmt.Sprintf("[]%s", parent)
		appender(slice)
	} else {
		Append(slice)
	}
	if sliceType == "struct" {
		allFields := make(map[string]struct {
			Value interface{}
			Count int
		})

		for i := 0; i < scopeLength; i++ {
			keys := reflect.ValueOf(scope[i]).MapKeys()
			for _, k := range keys {
				keyname := k.String()
				if _, ok := allFields[keyname]; !ok {
					allFields[keyname] = struct {
						Value interface{}
						Count int
					}{Value: scope[i], Count: 0}
				} else {
					existingValue := allFields[keyname].Value
					currentValue := scope[i]

					if compareObjects(existingValue, currentValue) {
						comparisonResult := compareObjectKeys(
							reflect.ValueOf(currentValue).MapKeys(),
							reflect.ValueOf(existingValue).MapKeys(),
						)
						if !comparisonResult {
							keyname = fmt.Sprintf("%s_%s", keyname, uuidv4())
							allFields[keyname] = struct {
								Value interface{}
								Count int
							}{Value: currentValue, Count: 0}
						}
					}
				}
				//allFields[keyname].Count++
			}
		}

		keys := reflect.ValueOf(allFields).MapKeys()
		structFields := make(map[string]interface{})
		omitempty := make(map[string]bool)
		for _, k := range keys {
			keyname := k.String()
			elem := allFields[keyname]

			structFields[keyname] = elem.Value
			omitempty[keyname] = elem.Count != scopeLength
		}
		parseStruct(depth+1, innerTabs, structFields, omitempty, flattens, example)
	} else if sliceType == "slice" {
		parseScope(scope[0], depth, flattens, example)
	} else {
		if flattens && depth >= 2 {
			appender(sliceType)
		} else {
			Append(sliceType)
		}
	}
}

func format(str string) string {
	str = formatNumber(str)

	sanitized := toProperCase(str)
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

func toProperCase(str string) string {
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

func formatScopeKeys(keys []string) []string {
	for i := range keys {
		keys[i] = format(keys[i])
	}
	return keys
}

func compareObjectKeys(itemAKeys, itemBKeys interface{}) bool {
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

func compareObjects(objectA, objectB interface{}) bool {
	typeObject := reflect.TypeOf(map[string]interface{}{})

	return reflect.TypeOf(objectA) == typeObject &&
		reflect.TypeOf(objectB) == typeObject
}

func getOriginalName(unique string) string {
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
func uuidv4() string {
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

func indent(tabs int) {
	for i := 0; i < tabs; i++ {
		// goCode.WriteString("\t")
		goCode += "\t"
	}
}

func Append(str string) {
	goCode += str
}

func indenter(tabs int) {
	for i := 0; i < tabs; i++ {
		stack[len(stack)-2] += "\t"
	}

}

func appender(str string) {
	// stack[len(stack)-2] += str
}

func main() {

	jsonStr := `{"name": "John", "age": 30, "city": "New York"}`

	typeName := "Person"

	goCode, err := jsonToGo(jsonStr, typeName, false, false, false)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(goCode)

}
