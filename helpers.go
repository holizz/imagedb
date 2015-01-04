package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"gopkg.in/mgo.v2/bson"
)

func listImages(w http.ResponseWriter, images []Image) {
	err := template.Must(template.New("").Parse(`<!doctype html>
	<ul>
	{{range .}}
	<li><a href="{{.Link}}"><img src="{{.RawLink}}"></a></li>
	{{end}}
	</ul>
	`)).Execute(w, images)
	if err != nil {
		panic(err)
	}
}

func addImage(image []byte, tags []string, originalName string) Image {
	c := session.DB("imagedb").C("images")

	if len(image) > int(math.Pow(2, 22)) {
		// the image is bigger than 4MB!
		panic(fmt.Errorf(`image too big: %d bytes`, len(image)))
	}

	mimeType := http.DetectContentType(image)

	storedImage := Image{
		ID:           bson.NewObjectId(),
		OriginalName: originalName,
		ContentType:  mimeType,
		Image:        image,
		Tags:         tags,
	}

	err := c.Insert(storedImage)
	if err != nil {
		panic(err)
	}

	return storedImage
}
