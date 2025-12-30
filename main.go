package main

import (
	"os"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"share/pkg/helper"
	"share/pkg/server"
	"share/pkg/backend"
)


type App struct {
	Port string
}


func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("error: file '.env' found :", err)
	}

	webapp := server.App{
		Port: "8080",
	}

	args := []string(os.Args[1:])
	if len(args) >= 1 {
		// go run share web
		if string(os.Args[1]) == "web" {
			go backend.PeriodicCleanExpiredShares()		// Goroutine to clean expired shares
			// go periodicCleanOrphansFiles()		// Goroutine to clean orphans files
			os.Setenv("DELETE_DB", "false")
			backend.CreateDatabase()
			webapp.Start()

		// go run share init
		// (= setup database at the first installation)
		} else if string(os.Args[1]) == "init" {
			fmt.Println("Looking for database")
			os.Setenv("DELETE_DB", "false")
			helper.DownloadStaticDependencies()
			backend.CreateDatabase()

		// go run share reset
		// (= reset database)
		} else if string(os.Args[1]) == "reset" {
			fmt.Println("Resetting database")
			os.Setenv("DELETE_DB", "true")
			backend.CreateDatabase()

		// go run share delete <shareId>
		} else if string(os.Args[1]) == "delete" {
			if len(args) > 1 {
				shareId := string(os.Args[2])
				fmt.Println("Deleting share '%s'", shareId)
				backend.DeleteShare(shareId)
			} else {
				fmt.Println("Please provide a share id")
			}

		// go run share backup
		} else if string(os.Args[1]) == "backup" {
			helper.BackupFile("sqlite.db")

		// go run share list
		} else if string(os.Args[1]) == "list" {
			backend.ListShareOpen()

		// go run share password <shareId>
		} else if string(os.Args[1]) == "password" {
			if len(args) > 1 {
				shareId := string(os.Args[2])
				backend.GetSharePassword(shareId)
			} else {
				fmt.Println("Please provide a share id")
			}

		// go run share help
		} else if string(os.Args[1]) == "help" {
			fmt.Println("Share is a web service that permit to securely share files and secrets to anyone")
			fmt.Println("")
			fmt.Println("Usage:")
			fmt.Println(" share web                  start web server")
			fmt.Println(" share init                 create database if not exists")
			fmt.Println(" share reset                delete database, it will be recreated next web server start")
			fmt.Println(" share backup               duplicate database (!does not backup shared files!)")
			fmt.Println(" share list                 get list of all the shares id")
			fmt.Println(" share password <shareId>   get the password of a share")
			fmt.Println(" share delete <shareId>     delete a share (also delete related shared files if any)")
			fmt.Println("")
			fmt.Println("https://github.com/ggtrd/share")

		// go run share <any wrong option>
		} else {
			fmt.Println("error: unknown command")
			fmt.Println("use 'share help' to display usage")
			fmt.Println("")
		}

	// go run share
	} else {
		fmt.Println("error: empty argument")
		fmt.Println("use 'share help' to display usage")
		fmt.Println("")
	}
}
