package helper

import (
	"time"
	// "fmt"
)


func GetNow() time.Time {
	now := time.Now().UTC()
	// fmt.Println(time.Parse(time.RFC822, now))

	return now
}


func GetTimeLayout() string {
	return "2006-01-02T15:04"
}