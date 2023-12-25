package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var data interface{}
var scope interface{}
var goCode strings.Builder
var tabs int

func jsonToGo(jsonStr string, typeName string, flatten bool, example bool, allOmitempty bool) string {
	
	seen := make(map[string][]string)
	stack := make([]string, 0)
	accumulator := ""
	innerTabs := 0
	parent := ""

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	typeName = format(typeName)
	goCode.WriteString("type " + typeName + " ")

	parseScope(data, &goCode, seen, &accumulator, &stack, flatten, example, allOmitempty, tabs, innerTabs, parent)

	return goCode.String()
}

func parseScope(scope interface{}, goCode *strings.Builder, seen map[string][]string, accumulator *string, stack *[]string, flatten bool, example bool, allOmitempty bool, tabs int, innerTabs int, parent string) {
	switch val := scope.(type) {
	case []interface{}:
		var sliceType string
		for _, item := range val {
			thisType := goType(item)
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
		if flatten && (sliceType == "struct" || sliceType == "slice") && innerTabs >= 2 {
			slice = "[]" + parent
			appender(slice, *accumulator)
		} else {
			slice = "[]"
			append(slice, goCode)
		}

		if sliceType == "struct" {
			allFields := make(map[string]struct {
				value interface{}
				count int
			})

			for _, item := range val {
				keys := getObjectKeys(item)
				for _, key := range keys {
					keyname := key.(string)
					if _, exists := allFields[keyname]; !exists {
						allFields[keyname] = struct {
							value interface{}
							count int
						}{
							value: item.(map[string]interface{})[keyname],
							count: 0,
						}
					} else {
						existingValue := allFields[keyname].value
						currentValue := item.(map[string]interface{})[keyname]

						if compareObjects(existingValue, currentValue) {
							comparisonResult := compareObjectKeys(
								getObjectKeys(currentValue),
								getObjectKeys(existingValue),
							)
							if !comparisonResult {
								keyname = keyname + "_" + uuidv4()
								allFields[keyname] = struct {
									value interface{}
									count int
								}{
									value: currentValue,
									count: 0,
								}
							}
						}
					}
					allFields[keyname].count++
				}
			}

			keys := getObjectKeys(allFields)
			structFields := make(map[string]interface{})
			omitempty := make(map[string]bool)
			for _, key := range keys {
				keyname := key.(string)
				elem := allFields[keyname]
				structFields[keyname] = elem.value
				omitempty[keyname] = elem.count != len(val)
			}

			parseStruct(tabs+1, innerTabs, structFields, omitempty, goCode, flatten, example, allOmitempty, seen, accumulator, stack, parent)
		} else if sliceType == "slice" {
			parseScope(val[0], goCode, seen, accumulator, stack, flatten, example, allOmitempty, tabs, innerTabs, parent)
		} else {
			if flatten && innerTabs >= 2 {
				appender(sliceType, accumulator)
			} else {
				append(sliceType, goCode)
			}
		}
	case map[string]interface{}:
		if flatten {
			if innerTabs >= 2 {
				appender(parent)
			} else {
				append(parent)
			}
		}
		parseStruct(tabs+1, innerTabs, val, nil, goCode, flatten, example, allOmitempty, seen, accumulator, stack, parent)
	default:
		if flatten && innerTabs >= 2 {
			appender(goType(val), accumulator)
		} else {
			append(goType(val), goCode)
		}
	}
}

func parseStruct(depth int, innerTabs int, scope map[string]interface{}, omitempty map[string]bool, goCode *strings.Builder, flatten bool, example bool, allOmitempty bool, seen map[string][]string, accumulator *string, stack *[]string, parent string) {
	if flatten {
		*stack = append(*stack, "")
	}

	seenTypeNames := make([]string, 0)

	if flatten && innerTabs >= 2 {
		parentType := "type " + parent
		scopeKeys := formatScopeKeys(getObjectKeys(scope))

		if seen[parent] != nil && compareObjectKeys(scopeKeys, seen[parent]) {
			*stack = (*stack)[:len(*stack)-1]
			return
		}
		seen[parent] = scopeKeys

		appender(parentType+" struct {\n", accumulator)
		innerTabs++
		keys := getObjectKeys(scope)
		for _, key := range keys {
			keyname := getOriginalName(key.(string))
			indenter(innerTabs, stack)
			typename := uniqueTypeName(format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			appender(typename+" ", accumulator)
			parent = typename
			parseScope(scope[keyname], goCode, seen, accumulator, stack, flatten, example, allOmitempty, depth, innerTabs, parent)
			appender('`json:"'+keyname, accumulator)
			if allOmitempty || (omitempty != nil && omitempty[keyname] == true) {
				appender(',omitempty', accumulator)
			}
			appender("\"`\n", accumulator)
		}
		indenter(innerTabs-1, stack)
		appender("}", accumulator)
	} else {
		append("struct {\n", goCode)
		tabs++
		keys := getObjectKeys(scope)
		for _, key := range keys {
			keyname := getOriginalName(key.(string))
			indent(tabs, goCode)
			typename := uniqueTypeName(format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)
			append(typename+" ", goCode)
			parent = typename
			parseScope(scope[keyname], goCode, seen, accumulator, stack, flatten, example, allOmitempty, depth, innerTabs, parent)
			append('`json:"'+keyname, goCode)
			if allOmitempty || (omitempty != nil && omitempty[keyname] == true) {
				append(',omitempty', goCode)
			}
			if example && scope[keyname] != "" && fmt.Sprintf("%T", scope[keyname]) != "map[string]interface {}" {
				append("\" example:\"" + fmt.Sprintf("%v", scope[keyname]), goCode)
			}
			append("\"`\n", goCode)
		}
		indent(tabs-1, goCode)
		append("}", goCode)
	}

	if flatten {
		*accumulator += (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]
	}
}



func format(str string) string {
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
		goCode += "\t"
	}
}

func append(str string) {
	goCode += str
}

func indenter(tabs int) {
	for i := 0; i < tabs; i++ {
		stack[len(stack)-1] += "\t"
	}
}

func appender(str string) {
	stack[len(stack)-1] += str
}
