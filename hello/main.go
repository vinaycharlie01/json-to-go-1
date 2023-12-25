package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

func jsonToGo(jsonStr string, typeName string, flatten bool, example bool, allOmitempty bool) string {
	var data interface{}
	var scope interface{}
	var goCode string
	var tabs int

	seen := make(map[string][]string)
	stack := []string{}
	accumulator := ""
	innerTabs := 0
	parent := ""

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}

	typeName = format(typeName)
	goCode += fmt.Sprintf("type %s ", typeName)

	parseScope(data)

	if flatten {
		goCode += accumulator
	}

	return goCode
}

func parseScope(scope interface{}, depth int) {
	switch s := scope.(type) {
	case map[string]interface{}:
		parseStruct(depth+1, innerTabs, s, nil)
	case []interface{}:
		parseSlice(depth+1, innerTabs, s)
	default:
		if flatten && depth >= 2 {
			appender(goType(scope))
		} else {
			append(goType(scope))
		}
	}
}

func parseStruct(depth, innerTabs int, scope map[string]interface{}, omitempty map[string]bool) {
	if flatten {
		stack = append(stack, strings.Repeat("\t", depth-2)+"\n")
	}

	seenTypeNames := []string{}

	if flatten && depth >= 2 {
		parentType := fmt.Sprintf("type %s", parent)
		scopeKeys := formatScopeKeys(reflect.ValueOf(scope).MapKeys())

		if seenKeys, ok := seen[parent]; ok && compareObjectKeys(scopeKeys, seenKeys) {
			stack = stack[:len(stack)-1]
			return
		}
		seen[parent] = scopeKeys

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
			parseScope(scope[key.Interface()], depth)
			appender(fmt.Sprintf(" `json:\"%s", keyname))
			if allOmitempty || (omitempty != nil && omitempty[keyname] {
				appender(",omitempty")
			}
			appender("\"`\n")
		}
		indenter(innerTabs - 1)
		appender("}")
	} else {
		append("struct {\n")
		tabs++
		keys := reflect.ValueOf(scope).MapKeys()
		for _, key := range keys {
			keyname := getOriginalName(key.String())
			indent(tabs)
			typename := uniqueTypeName(format(keyname), seenTypeNames)
			seenTypeNames = append(seenTypeNames, typename)

			append(typename + " ")
			parent = typename
			parseScope(scope[key.Interface()], depth)
			append(fmt.Sprintf(" `json:\"%s", keyname))
			if allOmitempty || (omitempty != nil && omitempty[keyname]) {
				append(",omitempty")
			}
			if example && scope[key.Interface()] != "" && reflect.TypeOf(scope[key.Interface()]).Kind() != reflect.Map {
				append(fmt.Sprintf("\" example:\"%v", scope[key.Interface()]))
			}
			append("\"\n")
		}
		indent(tabs - 1)
		append("}")
	}

	if flatten {
		accumulator += stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}
}

func parseSlice(depth, innerTabs int, scope []interface{}) {
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
	if flatten && depth >= 2 {
		slice = fmt.Sprintf("[]%s", parent)
		appender(slice)
	} else {
		append(slice)
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
					}{Value: scope[i][keyname], Count: 0}
				} else {
					existingValue := allFields[keyname].Value
					currentValue := scope[i][keyname]

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
				allFields[keyname].Count++
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
		parseStruct(depth+1, innerTabs, structFields, omitempty)
	} else if sliceType == "slice" {
		parseScope(scope[0], depth)
	} else {
		if flatten && depth >= 2 {
			appender(sliceType)
		} else {
			append(sliceType)
		}
	}
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

func uniqueTypeName(name string, seen []string) string {
	if !contains(seen, name) {
		return name
	}

	i := 0
	for {
		newName := fmt.Sprintf("%s%d", name, i)
		if !contains(seen, newName) {
			return newName
		}
		i++
	}
}

func format(str string) string {
	str = formatNumber(str)

	sanitized := toProperCase(str)
	if sanitized == "" {
		return "NAMING_FAILED"
	}

	return formatNumber(sanitized)
}

func formatNumber(str string) string {
	if str == "" {
		return ""
	} else if match, _ := regexp.MatchString(`^\d+$`, str); match {
		str = "Num" + str
	} else if str[0] >= '0' && str[0] <= '9' {
		numbers := map[string]string{
			"0": "Zero_", "1": "One_", "2": "Two_", "3": "Three_",
			"4": "Four_", "5": "Five_", "6": "Six_", "7": "Seven_",
			"8": "Eight_", "9": "Nine_",
		}
		str = numbers[str[0:1]] + str[1:]
	}

	return str
}

func goType(val interface{}) string {
	if val == nil {
		return "any"
	}

	switch v := val.(type) {
	case string:
		if matched, _ := regexp.MatchString(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(\+\d\d:\d\d|Z)$`, v); matched {
			return "time.Time"
		} else {
			return "string"
		}
	case float64:
		if v%1 == 0 {
			if v > -2147483648 && v < 2147483647 {
				return "int"
			} else {
				return "int64"
			}
		} else {
			return "float64"
		}
	case bool:
		return "bool"
	case map[string]interface{}:
		return "struct"
	case []interface{}:
		return "slice"
	default:
		return "any"
	}
}

func mostSpecificPossibleGoType(typ1, typ2 string) string {
	if strings.HasPrefix(typ1, "float") && strings.HasPrefix(typ2, "int") {
		return typ1
	} else if strings.HasPrefix(typ1, "int") && strings.HasPrefix(typ2, "float") {
		return typ2
	} else {
		return "any"
	}
}

func toProperCase(str string) string {
	if matched, _ := regexp.MatchString("^[_A-Z0-9]+$", str); matched {
		str = strings.ToLower(str)
	}

	commonInitialisms := []string{
		"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP",
		"HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA",
		"SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID",
		"URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS",
	}

	return regexp.MustCompile(`(^|[^a-zA-Z])([a-z]+)`).ReplaceAllStringFunc(str, func(m string) string {
		sep := m[0:1]
		frag := m[1:]
		if contains(commonInitialisms, strings.ToUpper(frag)) {
			return sep + strings.ToUpper(frag)
		} else {
			return sep + strings.ToUpper(frag[0:1]) + strings.ToLower(frag[1:])
		}
	})
}

func uuidv4() string {
	return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
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

func compareObjects(objectA, objectB interface{}) bool {
	typeObject := reflect.TypeOf(objectA).String()
	return typeObject == "map[string]interface {}"
}

func compareObjectKeys(itemAKeys, itemBKeys []reflect.Value) bool {
	lengthA := len(itemAKeys)
	lengthB := len(itemBKeys)

	if lengthA == 0 && lengthB == 0 {
		return true
	}

	if lengthA != lengthB {
		return false
	}

	for _, item := range itemAKeys {
		if !contains(itemBKeys, item) {
			return false
		}
	}
	return true
}

func formatScopeKeys(keys []reflect.Value) []string {
	var result []string
	for _, key := range keys {
		result = append(result, format(key.String()))
	}
	return result
}

func contains(slice interface{}, item interface{}) bool {
	switch reflect.TypeOf(slice).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(slice)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(s.Index(i).Interface(), item) {
				return true
			}
		}
	}
	return false
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-big" {
		var bufs []string
		bufs = append(bufs, "")
		for {
			var buf [8192]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				break
			}
			bufs = append(bufs, string(buf[:n]))
		}
		jsonStr := strings.Join(bufs, "")
		fmt.Println(jsonToGo(jsonStr, "AutoGenerated", true, false, false))
	} else {
		var buf [8192]byte
		var jsonStr string
		for {
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				break
			}
			jsonStr += string(buf[:n])
		}
		fmt.Println(jsonToGo(jsonStr, "AutoGenerated", true, false, false))
	}
}
