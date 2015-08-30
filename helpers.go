package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"text/template"

	"github.com/holizz/imagedb/db"
	"gopkg.in/mgo.v2/bson"
)

func listImages(w http.ResponseWriter, r *http.Request, session *db.Session, q string) {
	render(w, `
	{{define "title"}}List of images{{end}}
	{{define "body"}}

	<dom-module id="my-thingy">

		<template>
			<iron-ajax url="/api/search" params='{"q": "{{.q}}"}' last-response="{{"{{data}}"}}" auto></iron-ajax>
			<span>{{"{{data.Num}}"}}</span>

			<paper-drawer-panel>
				<paper-header-panel drawer>
					<paper-toolbar id="navheader">
						<span>Menu</span>
					</paper-toolbar>
					<paper-menu id="menu">

						<template is="dom-repeat" items="{{"{{data.Results}}"}}">
							<paper-item>
								<iron-image src="{{"{{rawLink(item.RawImage)}}"}}" preload sizing="contain" style="width: 100px; height: 100px"></iron-image>
							</paper-item>
						</template>

					</paper-menu>
				</paper-header-panel>

				<paper-header-panel main>
					<paper-toolbar id="mainheader">
						<form action="/search">
							<input type="search" name="q" value="{{.q}}" id="query">
							<input type="submit" value="Search">
						</form>
					</paper-toolbar>
					<iron-pages id="pages">

						<template is="dom-repeat" items="{{"{{data.Results}}"}}">
							<div>
								<a href="{{"{{link(item.ID)}}"}}">
									<iron-image src="{{"{{rawLink(item.RawImage)}}"}}" preload sizing="contain" style="width: 100%; height: 100%"></iron-image>
								</a>
							</div>
						</template>

					</iron-pages>
				</paper-header-panel>

			</paper-drawer-panel>

		</template>

		<script>
			Polymer({
				is: "my-thingy",
				rawLink: function(x) { return '/_image/' + x },
				link: function(x) { return '/image/' + x },
				ready: function () {
					var menu = this.$.menu
					var pages = this.$.pages

					menu.addEventListener('iron-select', function() {
						pages.select(this.selected)
					})

					document.onkeyup = function(e){
						var move = 0
						if (e.keyIdentifier === 'U+004B') {
							menu.selectPrevious()
						} else if (e.keyIdentifier === 'U+004A') {
							menu.selectNext()
						}
					}

					menu.select(0)
				}
			})
		</script>

		<style>
			/* fix height on images */
			#mainPanel,
			#mainContainer,
			html /deep/ paper-header-panel[main],
			html /deep/ iron-pages,
			html /deep/ iron-pages div {
				height: 100%;
			}
		</style>
	</dom-module>

	<my-thingy></my-thingy>
	{{end}}
	`, map[string]interface{}{
		"q": r.FormValue("q"),
	})
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
			<link rel="import" href="/bower_components/iron-pages/iron-pages.html">
			<link rel="import" href="/bower_components/iron-ajax/iron-ajax.html">
			<link rel="import" href="/bower_components/paper-menu/paper-menu.html">
			<link rel="import" href="/bower_components/paper-item/paper-item.html">
			<link rel="import" href="/bower_components/iron-image/iron-image.html">
			<link rel="import" href="/bower_components/paper-drawer-panel/paper-drawer-panel.html">
			<link rel="import" href="/bower_components/paper-header-panel/paper-header-panel.html">
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
