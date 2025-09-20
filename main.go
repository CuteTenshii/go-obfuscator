package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	patches = make(map[string]string)
)

func main() {
	inputRelative := flag.String("input", "", "Input project path")
	outputRelative := flag.String("output", "output.exe", "Output executable path")
	trimPaths := flag.Bool("trimpath", true, "Trim paths in the binary")
	buildVCS := flag.Bool("buildvcs", false, "Build with VCS info")
	ldFlags := flag.String("ldflags", "-s -w", "Additional ldflags for the Go build")
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

	goModPath := input + string(os.PathSeparator) + "go.mod"
	goModPackages := parseGoMod(goModPath)
	// Copy relevant files to a temporary directory to avoid modifying the original project
	processDirectory(input, tempFolder, goModPackages)
	processRenames(tempFolder, goModPackages)

	outputPath, err := filepath.Abs(*outputRelative)
	if err != nil {
		panic(err)
	}
	err = buildExecutable(buildExecutableOptions{
		ProjectPath: tempFolder,
		OutputPath:  outputPath,
		TrimPaths:   *trimPaths,
		BuildVCS:    *buildVCS,
		LdFlags:     strings.Split(*ldFlags, " "),
	})
	if err != nil {
		panic(err)
	}
	log.Println("Built executable at:", outputPath)
}

func processDirectory(inputPath, tempFolder string, goModPackages []string) {
	files, err := os.ReadDir(inputPath)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() {
			processDirectory(
				inputPath+string(os.PathSeparator)+fileName, tempFolder+string(os.PathSeparator)+fileName,
				goModPackages)
			continue
		}
		if !strings.HasSuffix(fileName, ".go") && fileName != "go.mod" && fileName != "go.sum" {
			continue
		}
		srcPath := inputPath + string(os.PathSeparator) + fileName
		dstPath := tempFolder + string(os.PathSeparator) + fileName
		err := copyFile(srcPath, dstPath)
		if err != nil {
			panic(err)
		}
		// Skip go.mod and go.sum files for patching
		if fileName == "go.mod" || fileName == "go.sum" {
			continue
		}

		log.Println("Processing file:", srcPath)
		// Process the file
		file, err := os.ReadFile(dstPath)
		patched := makeRenames(string(file), goModPackages)
		if err != nil {
			panic(err)
		}
		patches[dstPath] = patched
	}
}

func processRenames(tempFolder string, packages []string) {
	files, err := os.ReadDir(tempFolder)
	if err != nil {
		panic(err)
	}

	// Apply the patches to the files in the temporary directory
	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() {
			processRenames(tempFolder+string(os.PathSeparator)+fileName, packages)
			continue
		}
		if !strings.HasSuffix(fileName, ".go") && fileName != "go.mod" && fileName != "go.sum" {
			continue
		}
		// Skip go.mod and go.sum files for patching
		if fileName == "go.mod" || fileName == "go.sum" {
			continue
		}

		dstPath := tempFolder + string(os.PathSeparator) + file.Name()
		patched := applyRenames(patches[dstPath], packages)
		// Write the patched content back to the file
		err = os.WriteFile(dstPath, []byte(patched), 0644)
		if err != nil {
			panic(err)
		}
	}
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
			// Add the module name as well
			if strings.HasPrefix(line, "module ") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					packages = append(packages, fields[1])
				}
			}
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

type buildExecutableOptions struct {
	ProjectPath string
	OutputPath  string
	TrimPaths   bool
	BuildVCS    bool
	LdFlags     []string
}

func buildExecutable(options buildExecutableOptions) error {
	args := []string{"build"}
	if options.TrimPaths {
		args = append(args, "-trimpath")
	}
	if options.BuildVCS {
		args = append(args, "-buildvcs=false")
	}
	if len(options.LdFlags) > 0 {
		args = append(args, "-ldflags="+strings.Join(options.LdFlags, " "))
	}
	projectPath, err := filepath.Abs(options.ProjectPath)
	if err != nil {
		return err
	}
	outputPath, err := filepath.Abs(options.OutputPath)
	if err != nil {
		return err
	}
	// Append output path and project path to args
	args = append(args, "-o", outputPath, projectPath)
	log.Println("Building executable with command: go", strings.Join(args, " "))

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
