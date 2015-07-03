package main

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/holizz/imagedb/db"
	"gopkg.in/mgo.v2/bson"
)

func listImages(w http.ResponseWriter, r *http.Request, images []db.Image) {
	render(w, `
	{{define "title"}}List of images{{end}}
	{{define "body"}}
	<core-drawer-panel>
		<core-header-panel drawer>
			<core-toolbar id="navheader">
				<span>Menu</span>
			</core-toolbar>
			<core-menu>
				{{range .images}}
					<core-item>
						<core-image src="{{.RawLink}}" preload sizing="contain" style="width: 100px; height: 100px"></core-image>
					</core-item>
				{{end}}
			</core-menu>
		</core-header-panel>

		<core-header-panel main>
			<core-toolbar id="mainheader">
				<form action="/search">
					<input type="search" name="q" value="{{.q}}">
					<input type="submit" value="Search">
				</form>
			</core-toolbar>
			<core-pages>
				{{range .images}}
					<div>
						<a href="{{.Link}}">
							<core-image src="{{.RawLink}}" preload sizing="contain" style="width: 100%; height: 100%"></core-image>
						</a>
					</div>
				{{end}}
			</core-pages>
		</core-header-panel>

	</core-drawer-panel>

	<script>
		var menu = document.querySelector('core-menu')
		var pages = document.querySelector('core-pages')

		menu.addEventListener('core-select', function() {
			pages.selected = this.selected

			// fix height
			pages.style.height = (pages.parentElement.offsetHeight - pages.parentElement.firstElementChild.offsetHeight) + "px"
		})
		document.onkeyup = function(e){
			var move = 0
			if (e.keyIdentifier === 'U+004B') {
				move = -1
			} else if (e.keyIdentifier === 'U+004A') {
				move = 1
			}
			menu.selected = (menu.selected + move + menu.items.length) % menu.items.length
		}
	</script>

	<style>
	html /deep/ core-pages {
		overflow: hidden;
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
			<link rel="import" href="/bower_components/core-pages/core-pages.html">
			<link rel="import" href="/bower_components/core-menu/core-menu.html">
			<link rel="import" href="/bower_components/core-item/core-item.html">
			<link rel="import" href="/bower_components/core-image/core-image.html">
			<link rel="import" href="/bower_components/core-drawer-panel/core-drawer-panel.html">
			<link rel="import" href="/bower_components/core-header-panel/core-header-panel.html">
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

func parseQuery(s string) bson.M {
	tags := tagsFromString(s)
	return bson.M{
		"tags": bson.M{
			"$all": tags,
		},
	}
}
