package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"errors"
	"strings"
	"path"
	"path/filepath"
	"html/template"
	"net/http"
	"encoding/json"
	"strconv"
	"github.com/google/uuid"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/joho/godotenv"
)

type App struct {
	Port string
}

func main() {


	err := godotenv.Load()
	if err != nil {
		log.Println("No .env found :", err)
	}

	server := App{
		Port: env("PORT", "8080"),
	}
	

	args := []string(os.Args[1:])
	if len(args) >= 1 {
		// go run share web
		if string(os.Args[1]) == "web" {
			go periodicCleanExpiredShares()		// Goroutine to clean expired shares
			// go periodicCleanOrphansFiles()		// Goroutine to clean orphans files
			os.Setenv("DELETE_DB", "false")
			createDatabase()
			server.Start()

		// go run share init
		// (= setup database at the first installation)
		} else if string(os.Args[1]) == "init" {
			fmt.Println("Looking for database")
			os.Setenv("DELETE_DB", "false")
			createDatabase()

		// go run share reset
		// (= reset database)
		} else if string(os.Args[1]) == "reset" {
			fmt.Println("Resetting database")
			os.Setenv("DELETE_DB", "true")
			createDatabase()

		// go run share delete <shareId>
		} else if string(os.Args[1]) == "delete" {
			if len(args) > 1 {
				shareId := string(os.Args[2])
				fmt.Println("Deleting share '%s'", shareId)
				deleteShare(shareId)
			} else {
				fmt.Println("Please provide a share id")
			}

		// go run share backup
		} else if string(os.Args[1]) == "backup" {
			backupFile("sqlite.db")

		// go run share list
		} else if string(os.Args[1]) == "list" {
			listShareOpen()

		// go run share password <shareId>
		} else if string(os.Args[1]) == "password" {
			if len(args) > 1 {
				shareId := string(os.Args[2])
				getSharePassword(shareId)
			} else {
				fmt.Println("Please provide a share id")
			}

		// go run share help
		} else if string(os.Args[1]) == "help" {
			fmt.Println("Share is a web service that permit to securely share files and secrets to anyone")
			fmt.Println("")
			fmt.Println("Usage:")
			fmt.Println(" go run share web                  start web server")
			fmt.Println(" go run share init                 create database if not exists")
			fmt.Println(" go run share reset                delete database, it will be recreated next web server start")
			fmt.Println(" go run share backup               duplicate database (!does not backup shared files!)")
			fmt.Println(" go run share list                 get list of all the shares id")
			fmt.Println(" go run share password <shareId>   get the password of a share")
			fmt.Println(" go run share delete <shareId>     delete a share (also delete related shared files if any)")
			fmt.Println("")
			fmt.Println("https://github.com/ggtrd/share")

		// go run share <any wrong option>
		} else {
			fmt.Println("error: unknown command")
			fmt.Println("use 'go run share help' to display usage")
			fmt.Println("")
		}

	// go run share
	} else {
		fmt.Println("error: empty argument")
		fmt.Println("use 'go run share help' to display usage")
		fmt.Println("")
	}



}




func (a *App) Start() {

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	http.Handle("/", http.RedirectHandler("/secret", http.StatusSeeOther))		// Redirect to /secret by default

	http.Handle("/file", logReq(viewCreateFile))								// Form to create a share
	http.Handle("/file/shared", logReq(uploadFile))								// Confirmation + display the link of the share to the creator
	
	http.Handle("/secret", logReq(viewCreateSecret))							// Form to create a share
	http.Handle("/secret/shared", logReq(uploadSecret))							// Confirmation + display the link of the share to the creator

	http.Handle("/share/{id}", logReq(viewUnlockShare))							// Ask for password to unlock the share
	http.Handle("/share/unlock", logReq(unlockShare))							// Non browsable url - verify password to unlock the share
	http.Handle("/share/uploads/{id}/{file}", logReq(downloadFile))				// Download a shared file
	

	
	addr := fmt.Sprintf(":%s", a.Port)
	log.Printf(" web: starting app on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}




func env(key, adefault string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return adefault
	}
	return val
}




func logReq(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
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




func viewCreateFile(w http.ResponseWriter, r *http.Request) {

	// Generate a token that will permit to prevent unwanted record to database due to browse the upload URL without using the form
	// The trick is that this token is used from an hidden input on the HTML form, and if it's empty it means we're not using the form
	token := generatePassword()
	
	renderTemplate(w, "view.create.file.html", struct {
		TokenAvoidRefresh string
	}{
		TokenAvoidRefresh: token,
	})
}




func viewCreateSecret(w http.ResponseWriter, r *http.Request) {

	// Generate a token that will permit to prevent unwanted record to database due to browse the upload URL without using the form
	// The trick is that this token is used from an hidden input on the HTML form, and if it's empty it means we're not using the form
	token := generatePassword()

	renderTemplate(w, "view.create.secret.html", struct {
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
		PgpKeyPublic: getShareKeyPublic(shareId),
	})
}




func unlockShare(w http.ResponseWriter, r *http.Request)  {

	r.ParseForm()

	url:= r.Header.Get("Referer")
	idToUnlock := url[len(url)-36:] // Just get the last 36 char of the url because the IDs are 36 char length
	pgpMessageEncrypted := r.FormValue("pgpMessageEncrypted")

	// Decrypt PGP message
	// Using GopenPGP
	privateKey, err := crypto.NewKeyFromArmored(getShareKeyPrivate(idToUnlock))
	if err != nil {
		log.Println("err :", err)
		return
	}
	defer privateKey.ClearPrivateParams()
	pgp := crypto.PGP()
	decHandle, err := pgp.Decryption().DecryptionKey(privateKey).New()
	if err != nil {
		log.Println("err :", err)
		return
	}
	decrypted, err := decHandle.Decrypt([]byte(pgpMessageEncrypted), crypto.Armor)
	if err != nil {
		log.Println("err :", err)
		return
	}

	shareContentMap := getShareContent(idToUnlock)
	shareContentType := shareContentMap["type"]
	shareContentValue := shareContentMap["value"]


	shareOpenMap := getShareOpen(idToUnlock)
	shareCurrentOpen := shareOpenMap["currentopen"]
	shareMaxOpen := shareOpenMap["maxopen"]

	
	// Check if password match
	if decrypted.String() == getSharePassword(idToUnlock) {

		// Check if the share has not expired
		if shareCurrentOpen < shareMaxOpen {

			// Increment opened count
			updateShareOpen(idToUnlock)

			data := map[string]interface{}{
				// "sharePasswordHash": sharePasswordHash,
				"shareContentType": shareContentType,
				"shareContentValue": shareContentValue,
			}
			
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("err : could not marshal json: %s\n", err)
				return
			}
		
			w.Write(jsonData) // write JSON to JS


			// Check if this open is the last allowed and delete it, if it is (many 2 letters "i" words here ^^)
			shareOpenMap := getShareOpen(idToUnlock)
			shareCurrentOpen := shareOpenMap["currentopen"]
			shareMaxOpen := shareOpenMap["maxopen"]
			// if shareCurrentOpen >= shareMaxOpen {
			if shareCurrentOpen > shareMaxOpen {
				go deleteShare(idToUnlock)
			}



		} else {
			// Or delete the share because the maxopen has been reached
			go deleteShare(idToUnlock) // This should never comes here, but why don't leave this ?
		}
		

	} else {
		log.Println("err : password mismatch")
	}

}




func uploadSecret(w http.ResponseWriter, r *http.Request) {

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
	
	// --- Validation ---
	if len(secret) == 0 || len(secret) > 2500 {
		http.Error(w, "Secret must be between 1 and 2500 characters", http.StatusBadRequest)
		return
	}

	expTime, maxopen, err := validateExpirationAndMaxOpen(expiration, maxopenStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expirationFormatted := expTime.Format("2006-01-02T15:04")


	id := uuid.NewString()
	shared_id := uuid.NewString()
	uri := r.Header.Get("Referer")										// Entire path 'http://domain:port/node1/node2/etc.../'
	url:= path.Dir(uri)													// Only the 'http://domain:port' part
	link := strings.Join([]string{"/share/", shared_id}, "")

	// Create database entries
	createSecret(id, shared_id, secret, expirationFormatted, maxopen)

	// Display the confirmation
	renderTemplate(w, "view.confirm.share.html", struct {
		Link string				// To permit the user to click on it 
		Url string				// To permit the user to copy it
		Password string			// To permit the user to copy it
	}{
		Link: link,
		Url: url,
		Password: getSharePassword(shared_id),
	})
	
}




func uploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	// Ensure that a refresh of the page will not submit a new value in the database
	tokenAvoidRefresh := r.PostFormValue("TokenAvoidRefresh")
	if tokenAvoidRefresh == "" {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}
	expiration := r.PostFormValue("expiration")
	maxopenStr := r.PostFormValue("maxopen")

	expTime, maxopen, err := validateExpirationAndMaxOpen(expiration, maxopenStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expirationFormatted := expTime.Format("2006-01-02T15:04")


	id := uuid.NewString()
	shared_id := uuid.NewString()
	uri := r.Header.Get("Referer")										// Entire path 'http://domain:port/node1/node2/etc.../'
	url:= path.Dir(uri)													// Only the 'http://domain:port' part
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
	createFile(id, shared_id, filePath, expirationFormatted,maxopen)


	
	// Display the confirmation
	renderTemplate(w, "view.confirm.share.html", struct {
		Link string				// To permit the user to click on it 
		Url string				// To permit the user to copy it
		Password string			// To permit the user to copy it
	}{
		Link: link,
		Url: url,
		Password: getSharePassword(shared_id),
	})


}



func downloadFile(w http.ResponseWriter, r *http.Request) {

	url:= r.Header.Get("Referer")
	shareId := url[len(url)-36:]	// Just get the last 36 char of the url because the IDs are 36 char length
	shareContentMap := getShareContent(shareId)
	file := shareContentMap["value"]

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=" + file)

	http.ServeFile(w, r, file)
}