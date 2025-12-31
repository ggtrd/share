package backend

import (
	"log"
	"time"

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
