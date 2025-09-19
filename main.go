package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	inputRelative := flag.String("i", "", "Input project path")
	flag.Parse()
	if *inputRelative == "" {
		panic("Input project path is required")
	}

	tempFolder, err := os.MkdirTemp("/home/tenshii/Documents/go-obfuscator", "go-project-*")
	if err != nil {
		panic(err)
	}
	log.Println("Using temporary folder:", tempFolder)

	input, err := filepath.Abs(*inputRelative)
	if err != nil {
		panic(err)
	}
	files, err := os.ReadDir(input)
	if err != nil {
		panic(err)
	}

	goModPath := input + string(os.PathSeparator) + "go.mod"
	goModPackages := parseGoMod(goModPath)
	patches := make(map[string]string)
	// Copy relevant files to a temporary directory to avoid modifying the original project
	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(fileName, ".go") && fileName != "go.mod" && fileName != "go.sum" {
			continue
		}
		srcPath := input + string(os.PathSeparator) + fileName
		dstPath := tempFolder + string(os.PathSeparator) + fileName
		log.Println("Processing file:", srcPath)
		err := copyFile(srcPath, dstPath)
		if err != nil {
			panic(err)
		}

		// Process the file
		file, err := os.ReadFile(dstPath)
		patched := makeRenames(string(file), goModPackages)
		if err != nil {
			panic(err)
		}
		patches[dstPath] = patched
	}

	for _, file := range files {
		dstPath := tempFolder + string(os.PathSeparator) + file.Name()
		patched := applyRenames(patches[dstPath])
		// Write the patched content back to the file
		err = os.WriteFile(dstPath, []byte(patched), 0644)
		if err != nil {
			panic(err)
		}
	}

	outputPath := input + string(os.PathSeparator) + "output.exe"
	err = buildExecutable(tempFolder, outputPath)
	if err != nil {
		panic(err)
	}
	log.Println("Built executable at:", outputPath)
}

func parseGoMod(goModPath string) []string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		panic(err)
	}
	var packages []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "module ") ||
			strings.HasPrefix(line, "go ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			field := strings.TrimSpace(fields[0])
			if field == "require" || field == "replace" {
				field = strings.TrimSpace(fields[1])
			}
			if field == ")" || field == "(" {
				continue
			}
			field = strings.Replace(field, "require", "", 1)
			field = strings.Replace(field, "replace", "", 1)
			field = strings.Replace(field, "=>", "", 1)
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			packages = append(packages, field)
		}
	}

	return packages
}

func buildExecutable(projectPath, outputPath string) error {
	args := []string{
		"build", "-trimpath", "-buildvcs=false", `-ldflags=-s -w`,
		"-o", outputPath, projectPath,
	}
	log.Println("Building executable with command: go", strings.Join(args, " "))

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
