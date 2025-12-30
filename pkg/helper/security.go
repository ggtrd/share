package helper

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
	"strconv"
)


func ValidateExpirationAndMaxOpen(expiration string, maxopenStr string) (time.Time, int, error) {
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
func GeneratePassword() string {
	// get the secret key
	secret := os.Getenv("SHARE_SECRET_KEY")
	if secret == "" {
		log.Println("SHARE_SECRET_KEY missing")
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
