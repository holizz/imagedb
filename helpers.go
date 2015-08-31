package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/holizz/imagedb/db"
	"gopkg.in/mgo.v2/bson"
)

func listImages(w http.ResponseWriter, r *http.Request, session *db.Session, q string) {
	render(w, `
	{{define "title"}}List of images{{end}}
	{{define "body"}}

	<image-viewer></image-viewer>
	{{end}}
	`, map[string]interface{}{})
}

func addImage(session *db.Session, imageReader io.Reader, tags []string, originalName string) db.Image {
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

	rawImage, err := session.CreateRawImage()
	if err != nil {
		panic(err)
	}
	defer rawImage.Close()

	rawImage.SetContentType(mimeType)
	rawImage.Write(image[:n])

	storedImage := db.Image{
		ID:           bson.NewObjectId(),
		OriginalName: originalName,
		RawImage:     rawImage.Id().(bson.ObjectId).Hex(),
	}

	storedImage.SetTags(tags)

	err = session.Insert(storedImage)
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

func render(w io.Writer, tmpl string, context interface{}) {
	t, err := template.New("").Parse(`<!doctype html>
	<html>
		<head>
			<title>{{template "title" .}}</title>
			<script src="/bower_components/webcomponentsjs/webcomponents-lite.min.js"></script>
			<link rel="import" href="/bower_components/polymer/polymer.html">
			<link rel="import" href="/assets/polymer/image-viewer.html">
		</head>

		<body unresolved fullbleed>

			{{template "body" .}}

		</body>
	</html>
	`)

	err = template.Must(t.Parse(tmpl)).Execute(w, context)

	if err != nil {
		panic(err)
	}
}

func renderJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	err := e.Encode(data)
	if err != nil {
		panic(err)
	}
}

func parseQuery(s string) bson.M {
	if s == ":all" {
		return bson.M{}
	}

	if s == ":untagged" {
		return bson.M{
			"tags": []string{},
		}
	}

	tags := tagsFromString(s)
	return bson.M{
		"tags": bson.M{
			"$all": tags,
		},
	}
}
