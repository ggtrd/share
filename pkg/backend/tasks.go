package backend

import (
	"log"
	"os"
	"time"
	"database/sql"

	"github.com/go-co-op/gocron"

	"share/pkg/helper"
)


// Regularly check for all shares expiration date, and delete them if expired
func PeriodicCleanExpiredShares() {
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

			err := rows.Scan(&rowDataId, &rowDataExpiration)
			if err != nil {
				log.Println(" err:", err)
			}

			now := helper.GetNow()
			timeLayout := helper.GetTimeLayout()
			expiration, err := time.Parse(timeLayout, rowDataExpiration)
			if err != nil {
				log.Println(" err:", err)
			}

			// Delete share if its expiration date is before now
			if now.After(expiration) {
				// Set as Goroutine to avoid database crash due to too many connexion opened
				go DeleteShare(rowDataId)
			}
		}
    })
    task.StartAsync()

    // Prevent exit
    select {}
}


// Regularly delete orphan files (= when their related share doesn't exist anymore)
func PeriodicCleanOrphanFiles() {
	task := gocron.NewScheduler(time.UTC)
	task.Every(1).Minutes().Do(func() {
		log.Println("task: periodic clean of orphan files")

		entries, err := os.ReadDir("uploads")
		if err != nil {
			log.Println(" err:", err)
			return
		}

		db := openDatabase()
		defer db.Close()

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			shareId := entry.Name()

			// Skip freshly-created directories (upload-in-progress race guard)
			info, err := entry.Info()
			if err != nil {
				log.Println(" err:", err)
				continue
			}
			if time.Since(info.ModTime()) < 2 * time.Minute {
				continue
			}

			// Keep the directory only if a file row still references this share
			var rowDataShareId string
			err = db.QueryRow("SELECT share_id FROM file WHERE share_id = :share_id", shareId).Scan(&rowDataShareId)
			if err == sql.ErrNoRows {
				helper.DeletePath("uploads/" + shareId)
				log.Println("task: deleted orphan file dir", shareId)
			} else if err != nil {
				log.Println(" err:", err)
			}
		}
	})
	task.StartAsync()

	// Prevent exit
	select {}
}
