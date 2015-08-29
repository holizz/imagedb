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
	images, err := session.Find(parseQuery(q))
	if err != nil {
		panic(err)
	}

	render(w, `
	{{define "title"}}List of images{{end}}
	{{define "body"}}

	<paper-drawer-panel>
		<paper-header-panel drawer>
			<paper-toolbar id="navheader">
				<span>Menu</span>
			</paper-toolbar>
			<paper-menu>

				{{range .images}}
					<paper-item>
						<iron-image src="{{.RawLink}}" preload sizing="contain" style="width: 100px; height: 100px"></iron-image>
					</paper-item>
				{{end}}

			</paper-menu>
		</paper-header-panel>

		<paper-header-panel main>
			<paper-toolbar id="mainheader">
				<form action="/search">
					<input type="search" name="q" value="{{.q}}">
					<input type="submit" value="Search">
				</form>
			</paper-toolbar>
			<iron-pages>

				{{range .images}}
					<div>
						<a href="{{.Link}}">
							<iron-image src="{{.RawLink}}" preload sizing="contain" style="width: 100%; height: 100%"></iron-image>
						</a>
					</div>
				{{end}}

			</iron-pages>
		</paper-header-panel>

	</paper-drawer-panel>

	<script>
		var menu = document.querySelector('paper-menu')
		var pages = document.querySelector('iron-pages')

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
	{{end}}
	`, map[string]interface{}{
		"q":      r.FormValue("q"),
		"images": images,
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
			<link rel="import" href="/bower_components/iron-pages/iron-pages.html">
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

func renderJson(w io.Writer, data interface{}) {
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
