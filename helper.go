package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Generate a secure string
func generatePassword() string {

	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("err :", err)
	}

	password := base64.StdEncoding.EncodeToString(b)
	password = strings.Replace(password, "/", "", -1) // Remove unwanted char to avoid URL issues
	password = strings.Replace(password, "+", "", -1) // Remove unwanted char to avoid URL issues

	return password
}

// Delete a file or directory from filesystem
func createPath(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0700)
		if err != nil {
			log.Println("err :", err)
		}
	}
}

// Delete a file or directory from filesystem
func deletePath(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		log.Println("err :", err)
	}
}

// Copy/paste a file and automatically name it with current datetime
func backupFile(sourceFile string) {

	t := time.Now()
	now := fmt.Sprintf("%d-%02d-%02d_%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())

	// Open the source file
	source, err := os.Open(sourceFile)
	if err != nil {
		log.Println("err :", err)
	}
	defer source.Close()

	// Create the destination file
	destination, err := os.Create(sourceFile + "." + now)
	if err != nil {
		log.Println("err :", err)
	}
	defer destination.Close()

	// Copy the contents of source to destination file
	_, err = io.Copy(destination, source)
	if err != nil {
		log.Println("err :", err)
	}
}

// Check if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
