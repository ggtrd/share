package helper

import (
	"fmt"
	"log"
	"os"
	"io"
	"time"
	"net/http"
)


// Check if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}


// Delete a file or directory from filesystem
func CreatePath(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0700)
		if err != nil {
			log.Println("err :", err)
		}
	}
}


// Delete a file or directory from filesystem
func DeletePath(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		log.Println("err :", err)
	}
}


// Copy/paste a file and automatically name it with current datetime
func BackupFile(sourceFile string) {
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


// Download file from URL and store to given path
func DownloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Ensure HTTP is working
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy content
	_, err = io.Copy(out, resp.Body)
	return err
}


func GetFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println("error:", err)
	}
	return string(data)
}