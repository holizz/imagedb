package main

import (
	"log"

	"github.com/holizz/imagedb/db"
)

func main() {
	session := db.NewSessionFromEnv()

	err := session.Connect()
	if err != nil {
		panic(err)
	}
	defer session.Close()

	log.Println("Removing images tagged _delete...")

	n, err := session.RemoveAll("_delete")
	if err != nil {
		panic(err)
	}

	log.Printf("Finished. Deleted %d images", n)
}
