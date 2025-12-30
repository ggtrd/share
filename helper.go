package main

import (
	"fmt"
	"log"
	"os"
	"strings"
    "encoding/base64"
	"crypto/rand"
	"crypto/hmac"
	"crypto/sha256"
	"time"
	"io"
	"strconv"
)


func validateExpirationAndMaxOpen(expiration string, maxopenStr string) (time.Time, int, error) {
	// Parse expiration date
	expTime, err := time.Parse("2006-01-02T15:04", expiration)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("expiration date must be in format YYYY-MM-DDTHH:MM")
	}

	// Check if in the future
	if !expTime.After(time.Now()) {
		return time.Time{}, 0, fmt.Errorf("expiration date must be in the future")
	}

	// Parse maxopen into integer
	maxopen, err := strconv.Atoi(maxopenStr)
	if err != nil || maxopen < 1 || maxopen > 100 {
		return time.Time{}, 0, fmt.Errorf("max open must be a number between 1 and 100")
	}

	return expTime, maxopen, nil
}


// Generate a secure string
func generatePassword() string {

	// get the secret key
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		log.Println("SECRET_KEY missing")
	}

	// Generate random 32 bytes (256 bits)
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Println("Random generation error :", err)
	}

	// Optional : sign with HMAC
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(randomBytes)
	signed := h.Sum(nil)

	// Encdoing in base64 URL-safe
	token := base64.URLEncoding.EncodeToString(signed)

	// Remove padding `=`
	token = strings.TrimRight(token, "=")

	return token
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