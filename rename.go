package main

import (
	"encoding/base64"
	"math/rand"
	"regexp"
	"strings"
)

var (
	// Map of original names to new random names.
	// "originalName": "newName"
	renames = map[string]string{}
	// Track if base64 decoding function has been injected, and in which package
	base64DecodeInjected = map[string]string{}
	// List of constants. Used to make them variables
	constants = map[string]string{}

	// Example: func myFunction(
	funcRegex = regexp.MustCompile(`func (\w+)\(`)

	// Example: func myFunction(arg1 int, arg2 string) {
	funcWithArgsRegex = regexp.MustCompile(`func (\w+)\(.*\)`)

	// Example: var myVariable int
	varRegex = regexp.MustCompile(`var (\w+) `)

	// Example: var (
	//             var1 int
	//             var2 string
	//          )
	varBlockRegex = regexp.MustCompile(`var\s*\((?s:(.*?))\)`)

	// Example: const MyConst = 10
	constRegex = regexp.MustCompile(`const (\w+) `)

	// Example: const (
	//             const1 = 10
	//             const2 = "string"
	//          )
	constBlockRegex = regexp.MustCompile(`const\s*\((?s:(.*?))\)`)

	// Example: type MyType struct {
	typeRegex = regexp.MustCompile(`type (\w+) `)

	structItemRegex = regexp.MustCompile(`(\w+)\s+\w+`)

	// Example: myVar := 10
	varInFuncRegex = regexp.MustCompile(`(\w+) :=`)

	// Example: myVar, anotherVar := idk()
	varsInFuncWithCommaRegex = regexp.MustCompile(`([\w, ]+) :=`)

	// Example: // This is a comment
	commentRegex = regexp.MustCompile(`([^\s\t]+)?(// .*)`)

	// Example: "string"
	stringRegex = regexp.MustCompile(`(import\s*(\(\s*)?)?"(.*?)"`)

	// Examples: `string` or `raw string`
	rawStringRegex = regexp.MustCompile("`([^`]*)`")

	// Example: 45847
	numberRegex = regexp.MustCompile(`\b(\d+)\b`)
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func makeRenames(code string, packages []string) string {
	// Remove comments to avoid renaming within them
	comments := commentRegex.FindAllStringSubmatch(code, -1)
	for _, match := range comments {
		comment := strings.TrimSpace(match[2])
		code = strings.ReplaceAll(code, comment, "")
	}

	// Match function names
	matches := funcRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if originalName == "main" {
			continue
		}
		if _, exists := renames[originalName]; !exists {
			renames[originalName] = genNewName()
		}
	}

	// Match function names with arguments
	matches = funcWithArgsRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if originalName == "main" {
			continue
		}
		if _, exists := renames[originalName]; !exists {
			renames[originalName] = genNewName()
		}
	}

	// Match variable names
	matches = varRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if _, exists := renames[originalName]; !exists {
			renames[originalName] = genNewName()
		}
	}

	// Match variable names in var blocks
	multiVarMatches := varBlockRegex.FindAllStringSubmatch(code, -1)
	for _, match := range multiVarMatches {
		vars := match[1]
		varNames := extractVarNamesFromBlock(vars)
		for _, varName := range varNames {
			if _, exists := renames[varName]; !exists {
				renames[varName] = genNewName()
			}
		}
	}

	// Match constant names
	matches = constRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if _, exists := renames[originalName]; !exists {
			name := genNewName()
			renames[originalName] = name
			constants[originalName] = name
		}
	}

	// Match constant names in const blocks
	multiConstMatches := constBlockRegex.FindAllStringSubmatch(code, -1)
	for _, match := range multiConstMatches {
		consts := match[1]
		constNames := extractVarNamesFromBlock(consts)
		for _, constName := range constNames {
			if _, exists := renames[constName]; !exists {
				renames[constName] = genNewName()
			}
		}
	}

	// Match variable names within functions
	matches = varInFuncRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if originalName == "_" {
			continue
		}
		if _, exists := renames[originalName]; !exists {
			renames[originalName] = genNewName()
		}
	}

	// Match multiple variable names within functions (e.g., a, b := ...)
	multiVarMatches = varsInFuncWithCommaRegex.FindAllStringSubmatch(code, -1)
	for _, match := range multiVarMatches {
		vars := strings.Split(match[1], ",")
		for _, varName := range vars {
			varName = strings.Replace(varName, "for ", "", 1)
			varName = strings.Replace(varName, "if ", "", 1)
			varName = strings.Replace(varName, "else ", "", 1)
			varName = strings.TrimSpace(varName)
			if varName == "_" {
				continue
			}
			if _, exists := renames[varName]; !exists {
				renames[varName] = genNewName()
			}
		}
	}

	// Match type names
	matches = typeRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		originalName := match[1]
		if _, exists := renames[originalName]; !exists {
			renames[originalName] = genNewName()
		}
	}

	// Match struct item names
	//structMatches := regexp.MustCompile(`type\s+\w+\s+struct\s*{\s*((?:\s*\w+\s+\*?\[\]*\w+(?:\.\w+)?(?:\s+`+"`"+`[^`+"`"+`]*`+"`"+`)?\s*)+)}`).FindAllStringSubmatch(code, -1)
	//for _, structMatch := range structMatches {
	//	structBody := structMatch[1]
	//	items := structItemRegex.FindAllStringSubmatch(structBody, -1)
	//	for _, item := range items {
	//		originalName := item[1]
	//		if _, exists := renames[originalName]; !exists {
	//			renames[originalName] = genNewName()
	//		}
	//	}
	//}

	// Encode string literals in base64
	stringLiterals := stringRegex.FindAllStringSubmatch(code, -1)
	// List of internal Go packages. Those packages are NOT in the go.mod file
	internalPackages := []string{
		"fmt", "os", "io", "log", "net", "encoding", "bytes", "strings", "syscall", "unsafe",
		"regexp", "math", "time", "path", "slices", "flag", "strconv", "mime", "http",
		"database", "crypto", "errors", "runtime", "image",
	}
	for _, match := range stringLiterals {
		originalString := match[3]

		// Skip empty strings
		if originalString == "" || originalString == "\"\"" {
			continue
		}

		// Skip base64 encoding if it's an import statement
		if strings.Contains(match[0], "import ") {
			continue
		}
		for _, pkg := range internalPackages {
			if originalString == pkg || strings.HasPrefix(originalString, pkg+"/") {
				originalString = ""
				break
			}
		}
		isPackage := false
		for _, pkg := range packages {
			if originalString == pkg || strings.HasPrefix(originalString, pkg+"/") {
				isPackage = true
				break
			}
		}
		if isPackage {
			continue
		}
		if strings.HasPrefix(originalString, "`+") && strings.HasSuffix(originalString, "+`") {
			continue
		}

		encodedString := base64.StdEncoding.EncodeToString([]byte(originalString))
		newCode, fnName := injectBase64DecodeFunc(code)
		code = newCode
		code = strings.ReplaceAll(code, `"`+originalString+`"`, fnName+`("`+encodedString+`")`)
		code = addBase64EncodingImport(code, fnName)
	}

	// Encode raw string literals in base64
	rawStringLiterals := rawStringRegex.FindAllStringSubmatch(code, -1)
	for _, match := range rawStringLiterals {
		originalString := match[1]

		// Skip empty strings
		if originalString == "" {
			continue
		} else if strings.HasPrefix(originalString, "json:") || strings.HasPrefix(originalString, "yaml:") {
			continue
		}

		encodedString := base64.StdEncoding.EncodeToString([]byte(originalString))
		newCode, fnName := injectBase64DecodeFunc(code)
		code = newCode
		code = strings.ReplaceAll(code, "`"+originalString+"`", fnName+`("`+encodedString+`")`)
		code = addBase64EncodingImport(code, fnName)
	}

	return code
}

func doesConstOnlyContainStrings(block string) bool {
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		// Check if the line contains an '=' sign
		if !strings.Contains(line, "=") {
			return false
		}
		// Split by '=' and check if the value part is a string literal
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			return false
		}
		value := strings.TrimSpace(parts[1])
		if !(strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) &&
			!(strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`")) {
			return false
		}
	}
	return true
}

func applyRenames(code string, packages []string) string {
	code = strings.ReplaceAll(code, "const (", "var (")
	for original, newName := range renames {
		matchedPkg := false
		for _, pkg := range packages {
			if strings.Contains(pkg, original) {
				matchedPkg = true
				break
			}
		}
		if matchedPkg {
			continue
		}

		for originalConst := range constants {
			if original == originalConst {
				code = strings.ReplaceAll(code, "const "+original, "var "+newName)
			}
		}

		code = regexp.MustCompile(`\b`+original+`\b`).ReplaceAllString(code, newName)
	}
	return code
}

// genNewName generates a new random name for a given identifier
func genNewName() string {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	newName := make([]byte, 10)
	for i := range newName {
		newName[i] = characters[rand.Intn(len(characters))]
	}
	return string(newName)
}

func extractVarNamesFromBlock(block string) []string {
	var names []string
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		// Match variable name at the start of the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			name := fields[0]
			// Skip if it's not a valid identifier
			if regexp.MustCompile(`^[A-Za-z0-9_]\w*$`).MatchString(name) {
				names = append(names, name)
			}
		}
	}
	return names
}

func addBase64EncodingImport(code string, fnName string) string {
	if !strings.Contains(code, `"encoding/base64"`) && strings.Contains(code, "func "+fnName+"(s string) string") {
		if strings.Contains(code, "import (") {
			code = strings.Replace(code, "import (", "import (\n\t\"encoding/base64\"", 1)
		} else if strings.Contains(code, "import ") {
			code = strings.Replace(code, "import ", "import \"encoding/base64\"\n\nimport ", 1)
		} else {
			code = "import \"encoding/base64\"\n\n" + code
		}
	}

	return code
}

func injectBase64DecodeFunc(code string) (string, string) {
	funcName := genNewName()
	packageName := regexp.MustCompile(`package (\w+)`).FindStringSubmatch(code)
	if len(packageName) < 2 {
		return code, funcName
	}
	pkg := packageName[1]
	if base64DecodeInjected[pkg] != "" {
		return code, base64DecodeInjected[pkg]
	}

	// Inject the decode function only once per package
	if !strings.Contains(code, funcName) && !strings.Contains(code, "func "+funcName+"(s string) string") {
		decodeFunc := `
func ` + funcName + `(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(decoded)
}
`
		code = code + "\n" + decodeFunc
	}
	base64DecodeInjected[pkg] = funcName
	return code, funcName
}
