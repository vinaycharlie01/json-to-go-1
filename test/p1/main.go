package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

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

func main() {
	// Test the function
	result := formatNumber("123")
	fmt.Println(result) // Output: Num123

	result = formatNumber("456")
	fmt.Println(result) // Output: Four56

	result = formatNumber("abc")
	fmt.Println(result) // Output: abc
}
