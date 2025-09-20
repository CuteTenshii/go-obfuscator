package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func copyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.Create(dst)
	if err != nil {
		if strings.Contains(err.Error(), "The system cannot find the path specified") ||
			strings.Contains(err.Error(), "no such file or directory") {
			os.MkdirAll(filepath.Dir(dst), os.ModePerm)
			destFile, err = os.Create(dst)
			if err != nil {
				log.Println("Failed to create destination file after creating directories:", err)
				return err
			}
		} else {
			log.Println("Failed to create destination file:", err)
			return err
		}
	}
	defer destFile.Close()

	// Copy contents from source to destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Flush in case of buffered writer (not strictly needed here)
	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
