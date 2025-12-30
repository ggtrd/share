package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"strconv"
	"path/filepath"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/go-co-op/gocron"

	"github.com/ProtonMail/gopenpgp/v3/crypto"

)




// var dbFile string = "sqlite.db"
var dbFile string = filepath.Join("database", "sqlite.db")
var dbFileAuth string = filepath.Join("database", ".auth")

var rowFound    = "  db: records found from table:"
var rowNotFound = "  db: nothing found from table:"
var rowDeleted  = "  db: delete record from table:"



func createDatabase() {

	// first start             => create db if not exists, then run webserver          => DELETE_DB = false
	// init                    => create db if not exists                              => DELETE_DB = false
	// running without reset   => do nothing, then run webserver                       => DELETE_DB = false
	// reset                   => delete then create db (and create if if not exists)  => DELETE_DB = true


	createPath("database")


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
		if fileExists(dbFile) {
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
		if ! fileExists(dbFile) {
			
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
	// 	defer db.Close()


	return db
}




func createShare(id string, expirationGiven string, maxopenGiven int) {
	db := openDatabase()
	defer db.Close()


	t := time.Now()
	now := fmt.Sprintf("%d-%02d-%02dT%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())

	creation := sql.Named("creation", now)
	password := sql.Named("password", generatePassword())
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




func createFile(id string, shareId string, path string, expiration string, maxopen int) {
	db := openDatabase()
	defer db.Close()


	db.Exec("INSERT INTO file(id, path, share_id) values(:id, :path, :share_id)", id, path, shareId)


	createShare(shareId, expiration, maxopen)
}




func createSecret(id string, shareId string, text string, expiration string, maxopen int) {
	db := openDatabase()
	defer db.Close()


	db.Exec("INSERT INTO secret(id, text, share_id) values(:id, :text, :share_id)", id, text, shareId)


	createShare(shareId, expiration, maxopen)
}






// Get the content of a share
func getShareContent(shareId string) map[string]string {
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
func getSharePassword(shareId string) string {
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
func getShareKeyPublic(shareId string) string {
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
func getShareKeyPrivate(shareId string) string {
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
func getShareOpen(shareId string) map[string]string {
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
func updateShareOpen(shareId string) {
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
func deleteShare(shareId string) {
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
	deletePath("uploads/" + shareId)
	
}




// Get list of shares
func listShareOpen() {
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




// Set a task to run at a specific date
// Regularly check for all shares expiration date, and delete them if expired
func periodicCleanExpiredShares() {

	task := gocron.NewScheduler(time.UTC)
	task.Every(1).Minutes().Do(func() {
		log.Println("task: periodic clean of expired shares")


		db := openDatabase()
		defer db.Close()

	
		rows, err := db.Query("SELECT id, expiration FROM share")
		if err != nil {
			log.Println(" err:", err)
		}
		defer rows.Close()


		for rows.Next() {
			var rowDataId string
			var rowDataExpiration string

			err:= rows.Scan(&rowDataId, &rowDataExpiration)
			if err != nil {
				log.Println(" err:", err)
			}


			now := time.Now()
			timeLayout := "2006-01-02T15:04"
			expiration, err := time.Parse(timeLayout, rowDataExpiration)
			if err != nil {
				log.Println(" err:", err)
			}


			// Delete share if its expiration date is before now
			if now.After(expiration) {
				go deleteShare(rowDataId)	// Set as Goroutine to avoid database crash due to too many connexion opened
			}

		}
	
    })

    task.StartAsync()

    // Prevent exit
    select {}
}




// // Task to delete files front filesystem when their shares don't exist anymore (because maxopen value has been reached)
// func periodicCleanOrphansFiles() {

// 	task := gocron.NewScheduler(time.UTC)
// 	task.Every(1).Minutes().Do(func() {
// 		log.Println("task: periodic clean of orphans files")

// 		// Detect files from another function to be able to watch future uploads
// 		detectOrphansFiles()
		
//     })

//     task.StartAsync()

//     // Prevent exit
//     select {}
// }




// // Task to delete files from filesystem when their shares don't exist anymore (because maxopen value has been reached)
// func detectOrphansFiles() {

// 	dirUploads := "uploads/"


// 	files, err := ioutil.ReadDir(dirUploads)
//     if err != nil {
//         log.Println(" err:", err)
//     }



// 	db := openDatabase()
// 	defer db.Close()


// 	for _, file := range files {

// 		shareId := file.Name()											// Get the id from the directory name at 'upload/<id>' 
// 		shareIdPath := dirUploads + shareId


// 		// Get file creation date
// 		fileInfo, err := os.Stat(shareIdPath) 
// 		if err != nil {
// 			log.Println(" err:", err)
// 		}
// 		fileInfoTime := fileInfo.ModTime()								// The directory should never change, so modification date = creation date
// 		extendedExpirationDate := fileInfoTime.Add(24 * time.Hour)		// Create a "fake" extended expiration date for the file (this will permit to check if we consider the file can be deleted or not)

// 		now := time.Now()


// 		// Search for a database record corresponding to 'uploads/<id>/' directory
// 		row := db.QueryRow("SELECT id FROM share WHERE id = :share_id", shareId)
// 		var rowDataId string
// 		var readyToDelete bool
// 		switch err := row.Scan(&rowDataId); err {
// 			case sql.ErrNoRows:
// 				readyToDelete = true
// 			case nil:
// 				readyToDelete = false
// 			default:
// 				log.Println(" err:", err)
// 		}

		

// 		// Delete the file only if:
// 		//  - the share doesn't exist anymore
// 		//  - the creation date was a long time ago (which is defined by the 'extendedExpirationDate' variable)
// 		if (readyToDelete == true) && (now.After(extendedExpirationDate) == true) {
// 			log.Println("file: ready to delete:", shareIdPath, "created at", fileInfoTime, "expired at", extendedExpirationDate)
// 			go deletePath(shareIdPath) 										// Set as Goroutine to avoid database crash due to too many connexion opened
// 		}

// 	}
// }