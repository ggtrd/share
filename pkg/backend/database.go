package backend

import (
	"fmt"
	"log"
	"os"
	"time"
	"strconv"
	"path/filepath"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ProtonMail/gopenpgp/v3/crypto"

	"share/pkg/helper"
)


var dbFile string = filepath.Join("database", "sqlite.db")
var dbFileAuth string = filepath.Join("database", ".auth")

var rowFound    = "  db: records found from table:"
var rowNotFound = "  db: nothing found from table:"
var rowDeleted  = "  db: delete record from table:"


func CreateDatabase() {
	// first start             => create db if not exists, then run webserver          => DELETE_DB = false
	// init                    => create db if not exists                              => DELETE_DB = false
	// running without reset   => do nothing, then run webserver                       => DELETE_DB = false
	// reset                   => delete then create db (and create if if not exists)  => DELETE_DB = true

	helper.CreatePath("database")

	// Env var given from pseudo CLI
	var DELETE_DB, err = strconv.ParseBool(os.Getenv("DELETE_DB"))
	if err != nil {
		log.Println(" err:", err)
	}

	var query = `
	CREATE TABLE share (id text not null primary key, pgpkeypublic text, pgpkeyprivate text, password text, maxopen int, currentopen int, expiration text, creation text);
	DELETE FROM share;
	CREATE TABLE file (id text not null primary key, path text, share_id text, FOREIGN KEY(share_id) REFERENCES share(id));
	DELETE FROM file;
	CREATE TABLE secret (id text not null primary key, text text, share_id text, FOREIGN KEY(share_id) REFERENCES share(id));
	DELETE FROM secret;
	`

	// Reset database only if the user has decided to
	if DELETE_DB == true {

		// Check if file exists
		if helper.FileExists(dbFile) {
			os.Remove(dbFile)
		}
	
		// Open connexion
		db, err := sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Println(" err:", err)
		}
			defer db.Close()

		// Create tables
		_, err = db.Exec(query)
		if err != nil {
			log.Printf("%q: %s\n", err, query)
			return
		}

		log.Println("Database resetted")
	} else {

		// Check if file exists to create it if not
		if ! helper.FileExists(dbFile) {
			
			// Open connexion
			db, err := sql.Open("sqlite3", dbFile)
			if err != nil {
				log.Println(" err:", err)
			}
				defer db.Close()

			// Create tables
			_, err = db.Exec(query)
			if err != nil {
				log.Printf("%q: %s\n", err, query)
				return
			}

			log.Println("Database created")
		} else {
			log.Println("Database found")
		}

	}

	// Set database to be read/write only by owner (sqlite doesn't support user/auth)
	os.Chmod(dbFile, 0600)
}


func openDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Println(" err:", err)
	}

	return db
}


func CreateShare(id string, expirationGiven string, maxopenGiven int) {
	db := openDatabase()
	defer db.Close()

	t := time.Now()
	now := fmt.Sprintf("%d-%02d-%02dT%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())

	creation := sql.Named("creation", now)
	password := sql.Named("password", helper.GeneratePassword())
	maxopen := sql.Named("maxopen", maxopenGiven)
	currentopen := 0
	expiration := sql.Named("expiration", expirationGiven)

	pgp := crypto.PGP()
	keyGenHandle := pgp.KeyGeneration().AddUserId("share", id).New()
	keyPrivate, _ := keyGenHandle.GenerateKey()
	keyPublic, _ := keyPrivate.ToPublic()
	keyPrivateChain, _ := keyPrivate.Armor()
	keyPublicChain, _ := keyPublic.GetArmoredPublicKey()

	db.Exec("INSERT INTO share(id, password, pgpkeypublic, pgpkeyprivate, maxopen, currentopen, expiration, creation) values(:id, :password, :pgpkeypublic, :pgpkeyprivate, :maxopen, :currentopen, :expiration, :creation)", id, password, keyPublicChain, keyPrivateChain, maxopen, currentopen, expiration, creation)
}


func CreateShareFile(id string, shareId string, path string, expiration string, maxopen int) {
	db := openDatabase()
	defer db.Close()

	db.Exec("INSERT INTO file(id, path, share_id) values(:id, :path, :share_id)", id, path, shareId)

	CreateShare(shareId, expiration, maxopen)
}


func CreateShareSecret(id string, shareId string, text string, expiration string, maxopen int) {
	db := openDatabase()
	defer db.Close()

	db.Exec("INSERT INTO secret(id, text, share_id) values(:id, :text, :share_id)", id, text, shareId)

	CreateShare(shareId, expiration, maxopen)
}


// Get the content of a share
func GetShareContent(shareId string) map[string]string {
	db := openDatabase()
	defer db.Close()

	rowSecret := db.QueryRow("SELECT text FROM secret WHERE share_id = :share_id", shareId)
	var secretText string
	switch err := rowSecret.Scan(&secretText); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "secret")
		case nil:
			log.Println(rowFound, "secret")
		default:
			log.Println(" err:", err)
	}

	rowFile := db.QueryRow("SELECT path FROM file WHERE share_id = :share_id", shareId)
	var filePath string
	switch err := rowFile.Scan(&filePath); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "file")
		case nil:
			log.Println(rowFound, "file", filePath)
		default:
			log.Println(" err:", err)
	}

	if secretText != "" {
		return map[string]string{
			"type": "secret",
			"value": secretText,
		}

	} else if filePath != ""  {
		return map[string]string{
			"type": "file",
			"value": filePath,
		}

	} else {
		return map[string]string{
			"type": "none",
			"value": "none",
		}
	}
}


// Get the password of a share
func GetSharePassword(shareId string) string {
	db := openDatabase()
	defer db.Close()

	row := db.QueryRow("SELECT password FROM share WHERE id = :share_id", shareId)
	var rowData string
	switch err := row.Scan(&rowData); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "share")
		case nil:
			log.Println(rowFound, "share")
		default:
			log.Println(" err:", err)
	}
	
	return rowData
}


// Get the PGP public key of a share
func GetShareKeyPublic(shareId string) string {
	db := openDatabase()
	defer db.Close()

	row := db.QueryRow("SELECT pgpkeypublic FROM share WHERE id = :share_id", shareId)
	var rowData string
	switch err := row.Scan(&rowData); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "share")
		case nil:
			log.Println(rowFound, "share")
		default:
			log.Println(" err:", err)
	}
	
	return rowData
}


// Get the PGP private key of a share
func GetShareKeyPrivate(shareId string) string {
	db := openDatabase()
	defer db.Close()

	row := db.QueryRow("SELECT pgpkeyprivate FROM share WHERE id = :share_id", shareId)
	var rowData string
	switch err := row.Scan(&rowData); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "share")
		case nil:
			log.Println(rowFound, "share")
		default:
			log.Println(" err:", err)
	}
	
	return rowData
}


// Get the number of times a share has been opened
func GetShareOpen(shareId string) map[string]string {
	db := openDatabase()
	defer db.Close()

	row := db.QueryRow("SELECT currentopen, maxopen FROM share WHERE id = :share_id", shareId)
	var rowDataCurrentOpen string
	var rowDataMaxOpen string
	switch err := row.Scan(&rowDataCurrentOpen, &rowDataMaxOpen); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "share")
		case nil:
			log.Println(rowFound, "share")
		default:
			log.Println(" err:", err)
	}

	return map[string]string{
		"currentopen": rowDataCurrentOpen,
		"maxopen": rowDataMaxOpen,
	}
}


// Update the number of times a share has been opened
func UpdateShareOpen(shareId string) {
	db := openDatabase()
	defer db.Close()

	row := db.QueryRow("SELECT currentopen FROM share WHERE id = :share_id", shareId)
	var rowDataCurrentOpen string
	switch err := row.Scan(&rowDataCurrentOpen); err {
		case sql.ErrNoRows:
			log.Println(rowNotFound, "share")
		case nil:
			log.Println(rowFound, "share")
		default:
			log.Println(" err:", err)
	}

	// Increment the open (meaning it has been opened one time)
	currentopenInt, _ := strconv.Atoi(rowDataCurrentOpen)
	currentopen := currentopenInt + 1

	db.Exec("UPDATE share SET currentopen = :currentopen WHERE id = :share_id", currentopen, shareId)
}


// Delete a share and also its related secrets and files (and delete file from filesystem aswell)
func DeleteShare(shareId string) {
	db := openDatabase()
	defer db.Close()

	rowShare := db.QueryRow("DELETE FROM share WHERE id = :share_id", shareId)
	var rowShareData string
	switch err := rowShare.Scan(&rowShareData); err {
		case sql.ErrNoRows:
			log.Println(rowDeleted, "share", shareId)
		// case nil:
		// 	log.Println("Row found:", rowShareData)
		default:
			log.Println(" err:", err)
	}


	rowSecret := db.QueryRow("DELETE FROM secret WHERE share_id = :share_id", shareId)
	var rowSecretData string
	switch err := rowSecret.Scan(&rowSecretData); err {
		case sql.ErrNoRows:
			log.Println(rowDeleted, "secret", shareId)
		// case nil:
		// 	log.Println("Row found:", rowSecretData)
		default:
			log.Println(" err:", err)
	}

	rowFile := db.QueryRow("DELETE FROM file WHERE share_id = :share_id", shareId)
	var rowFileData string
	switch err := rowFile.Scan(&rowFileData); err {
		case sql.ErrNoRows:
			log.Println(rowDeleted, "file", shareId)
		// case nil:
		// 	log.Println("Row found:", rowFileData)
		default:
			log.Println(" err:", err)
	}

	// Delete the directory containing files of the share
	helper.DeletePath("uploads/" + shareId)	
}


// Get list of shares
func ListShareOpen() {
	db := openDatabase()
	defer db.Close()

	rows, err := db.Query("SELECT id, creation, expiration FROM share")
	if err != nil {
		log.Println(" err:", err)
	}
	defer rows.Close()

	var id string
	var creation string
	var expiration string
	for rows.Next() {
		 err:= rows.Scan(&id, &creation, &expiration)
		if err != nil {
			log.Println(" err:", err)
		}
		fmt.Println("ID:" + id + "; Created:" + creation + "; Expire:" + expiration)
	}
}

