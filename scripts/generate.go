// Package main provides a tool for generating Go code based on input data structures.
//
// This package reads column headers from various sources (CSV files, JSON files,
// or command-line arguments) and uses them to generate a Go struct with associated
// methods and functions. The generated code includes filter functions, getter
// functions, and sorting capabilities for the struct.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// TemplateData holds the column names for use in code generation.
type TemplateData struct {
	Columns []string
}

// capitalize returns the input string with its first letter capitalized.
//
// Example:
//
//	fmt.Println(capitalize("hello"))
//	// Output: Hello
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// toLower returns the input string converted to lowercase.
//
// Example:
//
//	fmt.Println(toLower("Hello"))
//	// Output: hello
func toLower(s string) string {
	return strings.ToLower(s)
}

// getCsvHeaders reads the first row of a CSV file and returns it as a slice of strings.
//
// Example:
//
//	headers, err := getCsvHeaders("data.csv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(headers)
//	// Output: [ID Name Age]
func getCsvHeaders(csvFilePath string) ([]string, error) {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read() // Read the first row as headers
	if err != nil {
		return nil, fmt.Errorf("error reading CSV headers: %w", err)
	}
	return headers, nil
}

// getJsonHeaders reads a JSON file and extracts the keys as headers.
// It supports both single JSON objects and arrays of objects.
//
// Example:
//
//	headers, err := getJsonHeaders("data.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(headers)
//	// Output: [id name age]
func getJsonHeaders(jsonFilePath string) ([]string, error) {
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening JSON file: %w", err)
	}
	defer file.Close()

	var jsonData interface{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&jsonData)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON file: %w", err)
	}

	var headers []string
	switch data := jsonData.(type) {
	case map[string]interface{}: // Single JSON object
		for key := range data {
			headers = append(headers, key)
		}
	case []interface{}: // JSON array (take keys from the first object if available)
		if len(data) > 0 {
			if firstObject, ok := data[0].(map[string]interface{}); ok {
				for key := range firstObject {
					headers = append(headers, key)
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported JSON structure: expecting object or array of objects")
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("no headers found in JSON file")
	}

	return headers, nil
}

// main is the entry point of the program. It parses command-line flags,
// reads headers from the specified source, and generates Go code based on those headers.
//
// Example:
//
//	go run main.go --columns "id,name,age" > model.go
//	go run main.go --csv-file data.csv > model.go
//	go run main.go --json-file data.json > model.go
func main() {
	columnsStr := flag.String("columns", "", "Comma-separated list of column headers")
	csvFile := flag.String("csv-file", "", "Path to a CSV file to read headers from")
	jsonFile := flag.String("json-file", "", "Path to a JSON file to read headers from")
	flag.Parse()

	var columns []string
	var err error

	if *csvFile != "" {
		columns, err = getCsvHeaders(*csvFile)
		if err != nil {
			fmt.Println("Error getting CSV headers:", err)
			os.Exit(1)
		}
		fmt.Println("Headers read from CSV file:", *csvFile)
	} else if *jsonFile != "" {
		columns, err = getJsonHeaders(*jsonFile)
		if err != nil {
			fmt.Println("Error getting JSON headers:", err)
			os.Exit(1)
		}
		fmt.Println("Headers read from JSON file:", *jsonFile)
	} else if *columnsStr != "" {
		columns = strings.Split(*columnsStr, ",")
		for i := range columns {
			columns[i] = strings.TrimSpace(columns[i])
		}
	} else {
		fmt.Println("Error: No column headers provided.")
		fmt.Println("Please provide column headers using either --columns, --csv-file, or --json-file flags.")

		fmt.Println("go run scripts/generate.go --columns \"id,title,slug\" > model.go")
		fmt.Println("go run scripts/generate.go --csv-file data/game_headers.csv > model.go")
		fmt.Println("go run scripts/generate.go --json-file data/game_headers.json > model.go")

		flag.Usage()
		os.Exit(1)
	}

	if len(columns) == 0 {
		fmt.Println("Error: No column headers could be determined.")
		fmt.Println("Please check your input file or command-line arguments.")
		os.Exit(1)
	}

	data := TemplateData{
		Columns: columns,
	}
	funcMap := template.FuncMap{
		"capitalize": capitalize,
		"toLower":    toLower,
		"backtick":   func() string { return "`" },
	}

	// The template (as a raw string literal) produces the generated Go code.
	const tpl = `
package main

import (
	"sort"
	"strconv"
	"strings"
)

// Item struct definition.
type Item struct {
{{- range .Columns }}
	{{ capitalize . }} string {{ backtick }}json:"{{ toLower . }}"{{ backtick }}
{{- end }}
}

// Columns returns the column names.
func (i Item) Columns() []string {
	return []string{
		{{- range .Columns }}
		"{{ toLower . }}",
		{{- end }}
	}
}

// Row returns the values of the item as a string slice.
func (i Item) Row() []string {
	return []string{
		{{- range .Columns }}
		i.{{ capitalize . }},
		{{- end }}
	}
}

// GetIndex returns the index for the item.
// Here, the first column is used as the index.
func (i Item) GetIndex() string {
	return i.{{ capitalize (index .Columns 0) }}
}

// ---- Getter Functions ----
{{- range .Columns }}
func Getters{{ capitalize . }}(i *Item) string {
	return i.{{ capitalize . }}
}
{{- end }}

// ---- Standard Filter Functions (case-sensitive) ----
{{- range .Columns }}
func Filter{{ capitalize . }}Contains(i *Item, s string) bool {
	return strings.Contains(i.{{ capitalize . }}, s)
}

func Filter{{ capitalize . }}StartsWith(i *Item, s string) bool {
	return strings.HasPrefix(i.{{ capitalize . }}, s)
}

func Filter{{ capitalize . }}Match(i *Item, s string) bool {
	return i.{{ capitalize . }} == s
}
{{- end }}

// ---- Case-Insensitive Helper Functions ----
func IContains(s, substr string) bool {
	return strings.Contains(strings.ToLower(substr), strings.ToLower(s))
}

func IMatch(s, match string) bool {
	return strings.EqualFold(s, match)
}

func hasPrefixCaseInsensitive(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return strings.EqualFold(s[:len(s)], prefix)
}

// ---- Case-Insensitive Filter Functions ----
{{- range .Columns }}
func Filter{{ capitalize . }}IContains(i *Item, s string) bool {
    return strings.Contains(strings.ToLower(i.{{ capitalize . }}), strings.ToLower(s))
}

func Filter{{ capitalize . }}IMatch(i *Item, s string) bool {
	return IMatch(i.{{ capitalize . }}, s)
}

func Filter{{ capitalize . }}IPrefix(i *Item, s string) bool {
	return hasPrefixCaseInsensitive(i.{{ capitalize . }}, s)
}
{{- end }}

// ---- Reduce Functions ----
func reduceCount(items Items) map[string]string {
	result := make(map[string]string)
	result["count"] = strconv.Itoa(len(items))
	return result
}

// ---- Type Definitions and Registration ----
type GroupedOperations struct {
	Funcs   registerFuncType
	GroupBy registerGroupByFunc
	Getters registerGettersMap
	Reduce  registerReduce
}

var Operations GroupedOperations

var RegisterFuncMap registerFuncType
var RegisterGroupBy registerGroupByFunc
var RegisterGetters registerGettersMap
var RegisterReduce registerReduce

func init() {
	RegisterFuncMap = make(registerFuncType)
	RegisterGroupBy = make(registerGroupByFunc)
	RegisterGetters = make(registerGettersMap)
	RegisterReduce = make(registerReduce)

	// Register standard match filters
	{{- range .Columns }}
	RegisterFuncMap["match-{{ toLower . }}"] = Filter{{ capitalize . }}Match
	{{- end }}

	// Register standard contains filters
	{{- range .Columns }}
	RegisterFuncMap["contains-{{ toLower . }}"] = Filter{{ capitalize . }}Contains
	{{- end }}

	// Register standard startswith filters
	{{- range .Columns }}
	RegisterFuncMap["startswith-{{ toLower . }}"] = Filter{{ capitalize . }}StartsWith
	{{- end }}

	// Register case-insensitive contains filters
	{{- range .Columns }}
	RegisterFuncMap["icontains-{{ toLower . }}"] = Filter{{ capitalize . }}IContains
	{{- end }}

	// Register case-insensitive match filters
	{{- range .Columns }}
	RegisterFuncMap["imatch-{{ toLower . }}"] = Filter{{ capitalize . }}IMatch
	{{- end }}

	// Register case-insensitive prefix filters
	{{- range .Columns }}
	RegisterFuncMap["iprefix-{{ toLower . }}"] = Filter{{ capitalize . }}IPrefix
	{{- end }}

	// Register getters
	{{- range .Columns }}
	RegisterGetters["{{ toLower . }}"] = Getters{{ capitalize . }}
	{{- end }}

	// Register groupby functions
	{{- range .Columns }}
	RegisterGroupBy["{{ toLower . }}"] = Getters{{ capitalize . }}
	{{- end }}

	// Register reduce functions
	RegisterReduce["count"] = reduceCount
}

// ---- Sorting Functions ----
func sortBy(items Items, sortingL []string) (Items, []string) {
	sortFuncs := map[string]func(i, j int) bool{
		{{- range .Columns }}
		"{{ toLower . }}": func(i, j int) bool { return items[i].{{ capitalize . }} < items[j].{{ capitalize . }} },
		"-{{ toLower . }}": func(i, j int) bool { return items[i].{{ capitalize . }} > items[j].{{ capitalize . }} },
		{{- end }}
	}
	
	for _, sortFuncName := range sortingL {
		sortFunc := sortFuncs[sortFuncName]
		sort.Slice(items, sortFunc)
	}
	
	// Gather available sort keys
	keys := []string{}
	for key := range sortFuncs {
		keys = append(keys, key)
	}
	return items, keys
}
`

	tmpl, err := template.New("model").Funcs(funcMap).Parse(tpl)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return
	}
}
