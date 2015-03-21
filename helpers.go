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
	<link rel="import" href="bower_components/core-pages/core-pages.html">
	<link rel="import" href="bower_components/core-menu/core-menu.html">
	<link rel="import" href="bower_components/core-item/core-item.html">
	<link rel="import" href="bower_components/core-image/core-image.html">
	<link rel="import" href="bower_components/core-drawer-panel/core-drawer-panel.html">
	<link rel="import" href="bower_components/core-header-panel/core-header-panel.html">

	<body unresolved fullbleed>

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

		menu.addEventListener('core-activate', function() {
			pages.selected = this.selected

			// fix height
			pages.style.height = (pages.parentElement.offsetHeight - pages.parentElement.firstElementChild.offsetHeight) + "px"
		})
	</script>

	<style>
	html /deep/ core-pages {
		overflow: hidden;
	}
	</style>

	</body>
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
