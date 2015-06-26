package main

import (
	"log"

	"github.com/holizz/imagedb/db"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	session := db.NewSessionFromEnv()

	err := session.Connect()
	if err != nil {
		panic(err)
	}
	defer session.Close()

	log.Println("Looking for duplicates...")

	dups := getDuplicates(session)

	log.Printf("Found %d unique images which have been duplicated.", len(dups))

	for _, dupGroup := range dups {
		tags := []string{}

		log.Println("Duplicate group:")

		for n, image := range dupGroup {
			// Collect all the tags

			tags = append(tags, image.Tags...)

			hash, err := image.Hash(session)
			if err != nil {
				panic(err)
			}

			log.Printf("  URL: %s Hash: %s Tags: %s", image.Link(), hash, image.TagsString())

			// Set tags to _delete

			dupGroup[n].SetTags([]string{"_delete"})
		}

		// Set tags of the first image to the collection

		dupGroup[0].SetTags(tags)

		log.Println("New duplicate group:")

		for _, image := range dupGroup {
			log.Printf("  Tags: %s", image.TagsString())
			session.UpdateId(image.ID.Hex(), image)
		}
	}

	log.Println("Finished")
}

func getDuplicates(session *db.Session) [][]db.Image {
	images, err := session.Find(bson.M{
		"tags": bson.M{
			"$nin": []string{"_delete"},
		},
	})
	if err != nil {
		panic(err)
	}

	duplicates := map[string][]db.Image{}

	for _, image := range images {
		hash, err := image.Hash(session)
		if err != nil {
			panic(err)
		}

		if _, ok := duplicates[hash]; !ok {
			duplicates[hash] = []db.Image{}
		}
		duplicates[hash] = append(duplicates[hash], image)
	}

	duplicateImages := [][]db.Image{}

	for _, dups := range duplicates {
		if len(dups) > 1 {
			duplicateImages = append(duplicateImages, dups)
		}
	}

	return duplicateImages
}
