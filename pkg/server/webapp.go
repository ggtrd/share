package server

import (
	"fmt"
	"log"
	"os"
	"io"
	"mime"
	"strconv"
	"strings"
	"path/filepath"
	"html"
	"html/template"
	"net/http"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/ProtonMail/gopenpgp/v3/crypto"

	"share/pkg/helper"
	"share/pkg/backend"
)


type App struct {
	Port string
}


func (a *App) Start() {

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	http.Handle("/", http.RedirectHandler("/password", http.StatusSeeOther))		// Redirect to /secret by default

	http.Handle("/about", logRequest(viewAbout))								// About

	http.Handle("/file", logRequest(viewCreateShareFile))						// Form to share a file
	http.Handle("/file/shared", logRequest(uploadShareFile))					// Confirmation + display the link of the share to the creator
	
	http.Handle("/password", logRequest(viewCreateShareSecretTextarea))			// Form to share a password
	http.Handle("/text", logRequest(viewCreateShareSecretRicheditor))			// Form to share a text
	http.Handle("/snippet", logRequest(viewCreateShareSecretRicheditor))		// Form to share a snippet
	http.Handle("/secret/shared", logRequest(uploadShareSecret))				// Confirmation + display the link of the share to the creator

	http.Handle("/share/{id}", logRequest(viewUnlockShare))						// Ask for password to unlock the share
	http.Handle("/share/unlock/{id}", logRequest(unlockShare))					// Non browsable url - verify password to unlock the share
	http.Handle("/share/uploads/{id}/{file}", logRequest(downloadShareFile))	// Download a shared file
	
	addr := fmt.Sprintf(":%s", a.Port)
	log.Printf(" web: starting app on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}


func logRequest(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(" web: %s", r.Header.Get("Referer"))
		f(w, r)
	})
}


func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	t, err := template.ParseGlob("templates/*.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("err : %s", err.Error()), 500)
		return
	}

	err = t.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("err : %s", err.Error()), 500)
		return
	}
}


func viewAbout(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "view.about.html", struct {
		Version string
	}{
		Version: helper.GetVersion(),
	})
}


func viewCreateShareFile(w http.ResponseWriter, r *http.Request) {
	// Generate a token that will permit to prevent unwanted record to database due to browse the upload URL without using the form
	// The trick is that this token is used from an hidden input on the HTML form, and if it's empty it means we're not using the form
	token := helper.GeneratePassword()
	
	renderTemplate(w, "view.create.file.html", struct {
		TokenAvoidRefresh string
	}{
		TokenAvoidRefresh: token,
	})
}


func viewCreateShareSecretTextarea(w http.ResponseWriter, r *http.Request) {
	// Generate a token that will permit to prevent unwanted record to database due to browse the upload URL without using the form
	// The trick is that this token is used from an hidden input on the HTML form, and if it's empty it means we're not using the form
	token := helper.GeneratePassword()

	renderTemplate(w, "view.create.secret.textarea.html", struct {
		TokenAvoidRefresh string
	}{
		TokenAvoidRefresh: token,
	})
}


func viewCreateShareSecretRicheditor(w http.ResponseWriter, r *http.Request) {
	// Generate a token that will permit to prevent unwanted record to database due to browse the upload URL without using the form
	// The trick is that this token is used from an hidden input on the HTML form, and if it's empty it means we're not using the form
	token := helper.GeneratePassword()

	renderTemplate(w, "view.create.secret.richeditor.html", struct {
		TokenAvoidRefresh string
	}{
		TokenAvoidRefresh: token,
	})
}


func viewUnlockShare(w http.ResponseWriter, r *http.Request) {
	shareId := r.PathValue("id")

	renderTemplate(w, "view.unlock.share.html", struct {
		ShareId string
		PgpKeyPublic string
	}{
		ShareId: shareId,
		PgpKeyPublic: backend.GetShareKeyPublic(shareId),
	})
}


func unlockShare(w http.ResponseWriter, r *http.Request)  {
	r.ParseForm()

	idToUnlock := r.PathValue("id")
	pgpMessageEncrypted := r.FormValue("pgpMessageEncrypted")

	// Every unlock attempt is logged with its origin, to be able to identify automated openings (mail scanners, link previews...)
	logUnlock := func(result string) {
		log.Printf(" unlock: share=%s result=%s ip=%s ua=%q", idToUnlock, result, clientIP(r), r.UserAgent())
	}

	// Decrypt PGP message
	// Using GopenPGP
	privateKey, err := crypto.NewKeyFromArmored(backend.GetShareKeyPrivate(idToUnlock))
	if err != nil {
		logUnlock("unknown-share")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer privateKey.ClearPrivateParams()
	pgp := crypto.PGP()
	decHandle, err := pgp.Decryption().DecryptionKey(privateKey).New()
	if err != nil {
		log.Println("err :", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	decrypted, err := decHandle.Decrypt([]byte(pgpMessageEncrypted), crypto.Armor)
	if err != nil {
		logUnlock("invalid-message")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Check if password match
	if decrypted.String() != backend.GetSharePassword(idToUnlock) {
		logUnlock("password-mismatch")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	shareContentMap := backend.GetShareContent(idToUnlock)
	shareContentType := shareContentMap["type"]
	shareContentValue := shareContentMap["value"]

	switch shareContentType {

	// For a file, the view is only counted when the file is actually downloaded (see downloadShareFile).
	// Unlocking only checks that views are left and gives a single use download token.
	case "file":
		shareOpenMap := backend.GetShareOpen(idToUnlock)
		shareCurrentOpen, _ := strconv.Atoi(shareOpenMap["currentopen"])
		shareMaxOpen, _ := strconv.Atoi(shareOpenMap["maxopen"])
		if shareCurrentOpen >= shareMaxOpen {
			logUnlock("no-view-left")
			backend.DeleteShare(idToUnlock)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		setDownloadTokenCookie(w, r, idToUnlock)

	// For a secret, displaying the content is the view
	case "secret":
		if !backend.TryIncrementShareOpen(idToUnlock) {
			logUnlock("no-view-left")
			backend.DeleteShare(idToUnlock)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Sanitize at display time, so that secrets stored before sanitization existed are covered too
		shareContentValue = sanitizeSecretHTML(shareContentValue)

	default:
		logUnlock("no-content")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"shareContentType": shareContentType,
		"shareContentValue": shareContentValue,
	})
	if err != nil {
		log.Printf("err : could not marshal json: %s\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	logUnlock("ok-" + shareContentType)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

	// Delete the secret if this was its last allowed view
	if shareContentType == "secret" {
		deleteShareIfLastView(idToUnlock)
	}
}


// Delete a share (and its uploaded files if any) if all its views have been consumed
func deleteShareIfLastView(shareId string) {
	shareOpenMap := backend.GetShareOpen(shareId)
	shareCurrentOpen, _ := strconv.Atoi(shareOpenMap["currentopen"])
	shareMaxOpen, _ := strconv.Atoi(shareOpenMap["maxopen"])
	if shareCurrentOpen >= shareMaxOpen {
		backend.DeleteShare(shareId)
		helper.DeletePath(filepath.Join("uploads", filepath.Base(shareId)))
	}
}


// Real client IP, taking a reverse proxy into account
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.RemoteAddr
}


// Policy allowing the formatting produced by the rich editor, without scripts, event handlers, etc.
var secretPolicy = bluemonday.UGCPolicy()

func sanitizeSecretHTML(value string) string {
	return secretPolicy.Sanitize(value)
}


// Plain text secrets (passwords) are escaped so that they are displayed exactly as typed
func plainTextToHTML(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(html.EscapeString(value), "\n", "<br>")
}


func uploadShareSecret(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// Ensure that a refresh of the page will not submit a new value in the database
	tokenAvoidRefresh := r.PostFormValue("TokenAvoidRefresh")
	if tokenAvoidRefresh == "" {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}
	secret := r.PostFormValue("mySecret")
	expiration := r.PostFormValue("expiration")
	maxopenStr := r.PostFormValue("maxopen")
	
	// Validation
	if len(secret) == 0 || len(secret) > 999999999 {
		http.Error(w, "Secret must be between 1 and 999999999 characters", http.StatusBadRequest)
		return
	}

	expirationValidated, maxopen, err := helper.ValidateExpirationAndMaxOpen(expiration, maxopenStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expirationFormatted := expirationValidated.Format(helper.GetTimeLayout())

	// Only the rich editor sends HTML, anything else is plain text and must be displayed as typed
	if r.PostFormValue("format") == "html" {
		secret = sanitizeSecretHTML(secret)
	} else {
		secret = plainTextToHTML(secret)
	}

	id := uuid.NewString()
	shared_id := uuid.NewString()
	url := "https://" + r.Host
	link := strings.Join([]string{"/share/", shared_id}, "")

	// Create database entries
	backend.CreateShareSecret(id, shared_id, secret, expirationFormatted, maxopen)

	// Display the confirmation
	renderTemplate(w, "view.confirm.share.html", struct {
		Link string				// To permit the user to click on it 
		Url string				// To permit the user to copy it
		Password string			// To permit the user to copy it
	}{
		Link: link,
		Url: url,
		Password: backend.GetSharePassword(shared_id),
	})
}


func uploadShareFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	// Ensure that a refresh of the page will not submit a new value in the database
	tokenAvoidRefresh := r.PostFormValue("TokenAvoidRefresh")
	if tokenAvoidRefresh == "" {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}
	expiration := r.PostFormValue("expiration")
	maxopenStr := r.PostFormValue("maxopen")

	expirationValidated, maxopen, err := helper.ValidateExpirationAndMaxOpen(expiration, maxopenStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expirationFormatted := expirationValidated.Format(helper.GetTimeLayout())


	id := uuid.NewString()
	shared_id := uuid.NewString()
	url := "https://" + r.Host
	link := strings.Join([]string{"/share/", shared_id}, "")

	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		// log.Println("err : can't retrieve file", file)
		log.Println("err :", err)
		return
	}
	defer file.Close()
	// log.Printf("Uploaded file: %+v\n", handler.Filename)
	// log.Printf("File size: %+v\n", handler.Size)
	// log.Printf("MIME header: %+v\n", handler.Header)


	// Create destination directory root
	dirUploads := "uploads/"
	if _, err := os.Stat(dirUploads); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dirUploads, 0700)
		if err != nil {
			log.Println("err :", err)
		}
	}

	// Create destination directory of the share
	dir := dirUploads + shared_id
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			log.Println("err :", err)
		}
	}

	// Create file
	filePath := filepath.Join(dir, filepath.Base(handler.Filename))
	dst, err := os.Create(filePath)
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stat, err := dst.Stat()
	if err != nil {
		log.Fatal(err.Error())
	}

	size, _ := strconv.Atoi(strconv.FormatInt(stat.Size(), 10))
	fmt.Println(size)

	// Create database entries
	backend.CreateShareFile(id, shared_id, filePath, expirationFormatted,maxopen)
	
	// Display the confirmation
	renderTemplate(w, "view.confirm.share.html", struct {
		Link string				// To permit the user to click on it 
		Url string				// To permit the user to copy it
		Password string			// To permit the user to copy it
	}{
		Link: link,
		Url: url,
		Password: backend.GetSharePassword(shared_id),
	})
}


func downloadShareFile(w http.ResponseWriter, r *http.Request) {
	shareId := r.PathValue("id")
	requestedFile := r.PathValue("file")

	cookie, err := r.Cookie(downloadCookieName)
	if err != nil || !consumeDownloadToken(cookie.Value, shareId) {
		log.Printf(" download: share=%s result=unauthorized ip=%s ua=%q", shareId, clientIP(r), r.UserAgent())
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	shareContentMap := backend.GetShareContent(shareId)
	if shareContentMap["type"] != "file" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	filePath := shareContentMap["value"]

	filePathClean := filepath.Clean(filePath)
	if !strings.HasPrefix(filePathClean, "uploads"+string(os.PathSeparator)) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	fileName := filepath.Base(filePathClean)
	if fileName != requestedFile {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// The download is what consumes a view of a file share
	if !backend.TryIncrementShareOpen(shareId) {
		log.Printf(" download: share=%s result=no-view-left ip=%s ua=%q", shareId, clientIP(r), r.UserAgent())
		deleteShareIfLastView(shareId)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	log.Printf(" download: share=%s result=ok ip=%s ua=%q", shareId, clientIP(r), r.UserAgent())

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	http.ServeFile(w, r, filePathClean)

	// Delete the share and its file once the last allowed download has been served
	deleteShareIfLastView(shareId)
}
