package main

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"

	"github.com/holizz/imagedb/db"
	"gopkg.in/mgo.v2/bson"
)

func addImage(session *db.Session, imageReader io.Reader, tags []string, originalName string) db.Image {
	startOfImage := make([]byte, 512)
	_, err := imageReader.Read(startOfImage)
	if err != nil {
		panic(err)
	}
	mimeType := http.DetectContentType(startOfImage)

	rawImage, err := session.CreateRawImage()
	if err != nil {
		panic(err)
	}
	defer rawImage.Close()

	rawImage.SetContentType(mimeType)
	_, err = rawImage.Write(startOfImage)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(rawImage, imageReader)
	if err != nil {
		panic(err)
	}

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

func render(w io.Writer, tmpl string, context interface{}) {
	t, err := template.New("").Parse(`<!doctype html>
	<html>
		<head>
			<title>{{template "title" .}}</title>
			<script src="/bower_components/webcomponentsjs/webcomponents-lite.min.js"></script>
			<link rel="import" href="/bower_components/polymer/polymer.html">
			<link rel="import" href="/assets/polymer/image-db.html">
		</head>

		<body unresolved>

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
