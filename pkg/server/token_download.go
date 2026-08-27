package server

import (
	"net/http"
	"sync"
	"time"

	"share/pkg/helper"
)

type downloadToken struct {
	shareId string
	expires time.Time
}

var (
	downloadTokensMu sync.Mutex
	downloadTokens   = map[string]downloadToken{}
)

const downloadTokenTTL = 60 * time.Second
const downloadCookieName = "share_token_download"


// Generates a token linked to the shareId, sent to the client only after a successful password unlock
func generateDownloadToken(shareId string) string {
	token := helper.GeneratePassword()
	downloadTokensMu.Lock()
	downloadTokens[token] = downloadToken{shareId: shareId, expires: time.Now().Add(downloadTokenTTL)}
	downloadTokensMu.Unlock()

	return token
}


// Validates a single use token for shareId and deletes it
func consumeDownloadToken(token, shareId string) bool {
	if token == "" {
		return false
	}

	downloadTokensMu.Lock()
	defer downloadTokensMu.Unlock()

	t, ok := downloadTokens[token]
	if !ok {
		return false
	}

	if time.Now().After(t.expires) {
		delete(downloadTokens, token) // clean up expired tokens
		return false
	}

	return t.shareId == shareId
}


// Store token in a HttpOnly cookie paired with shareId
func setDownloadTokenCookie(w http.ResponseWriter, r *http.Request, shareId string) {
	http.SetCookie(w, &http.Cookie{
		Name:     downloadCookieName,
		Value:    generateDownloadToken(shareId),
		Path:     "/share/uploads/" + shareId + "/",
		MaxAge:   int(downloadTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   helper.IsSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
	})
}