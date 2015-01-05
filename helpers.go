package main

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"strings"

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

func addImage(imageReader io.Reader, tags []string, originalName string) Image {
	c := session.DB("imagedb").C("images")

	image := make([]byte, int(math.Pow(2, 22))+1)
	n, err := io.ReadFull(imageReader, image)
	if err != nil && err != io.ErrUnexpectedEOF {
		panic(err)
	}

	if n > int(math.Pow(2, 22)) {
		// the image is bigger than 4MB!
		panic(fmt.Errorf(`image too big: %d bytes`, n))
	}

	mimeType := http.DetectContentType(image[:n])

	storedImage := Image{
		ID:           bson.NewObjectId(),
		OriginalName: originalName,
		ContentType:  mimeType,
		Image:        image[:n],
		Tags:         tags,
	}

	err = c.Insert(storedImage)
	if err != nil {
		panic(err)
	}

	return storedImage
}

func tagsFromString(s string) []string {
	tags := []string{}
	for _, t := range strings.Split(s, " ") {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
