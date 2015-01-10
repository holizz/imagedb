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

func listImages(w http.ResponseWriter, r *http.Request, images []Image) {
	err := template.Must(template.New("").Parse(`<!doctype html>
	<form action="/search">
	<input type="search" name="q" value="{{.q}}">
	<input type="submit" value="Search">
	</form>
	<ul>
	{{range .images}}
	<li><a href="{{.Link}}"><img src="{{.RawLink}}"></a></li>
	{{end}}
	</ul>
	`)).Execute(w, map[string]interface{}{
		"q":      r.FormValue("q"),
		"images": images,
	})
	if err != nil {
		panic(err)
	}
}

func addImage(imageReader io.Reader, tags []string, originalName string) Image {
	c := session.DB("imagedb").C("images")
	d := session.DB("imagedb").C("raw_images")

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

	rawImage := RawImage{
		ID:          bson.NewObjectId(),
		ContentType: mimeType,
		Image:       image[:n],
	}

	err = d.Insert(rawImage)
	if err != nil {
		panic(err)
	}

	storedImage := Image{
		ID:           bson.NewObjectId(),
		OriginalName: originalName,
		Tags:         tags,
		RawImage:     rawImage.ID.Hex(),
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
