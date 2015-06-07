package main

import (
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/justinas/alice"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	session *mgo.Session
)

func main() {
	var err error

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	mongoHost := os.Getenv("MONGODB")
	if len(mongoHost) == 0 {
		mongoHost = "localhost:27017"
	}

	session, err = mgo.Dial(mongoHost)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	common := alice.New(handleLogging)

	http.Handle("/bower_components/", common.Then(http.StripPrefix("/bower_components/", http.FileServer(http.Dir("bower_components")))))

	http.Handle("/", common.ThenFunc(handleRoot))
	http.Handle("/all", common.ThenFunc(handleAll))
	http.Handle("/_image/", common.ThenFunc(handleRawImage))
	http.Handle("/image/", common.ThenFunc(handleImage))
	http.Handle("/tags", common.ThenFunc(handleTagsList))
	http.Handle("/untagged", common.ThenFunc(handleUntagged))
	http.Handle("/download", common.ThenFunc(handleDownload))
	http.Handle("/upload", common.ThenFunc(handleUpload))
	http.Handle("/search", common.ThenFunc(handleSearch))
	http.Handle("/duplicates", common.ThenFunc(handleDuplicates))

	log.Fatalln(http.ListenAndServe(":"+port, nil))
}

func handleAll(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(nil).All(&images)
		if err != nil {
			panic(err)
		}

		listImages(w, r, images)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleRawImage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		hexId := bson.ObjectIdHex(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])

		gridfs := session.DB("imagedb").GridFS("raw_images2")

		image, err := gridfs.OpenId(hexId)
		if err != nil {
			panic(err)
		}
		defer image.Close()

		w.Header()["Content-type"] = []string{image.ContentType()}
		_, err = io.Copy(w, image)
		if err != nil {
			panic(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleImage(w http.ResponseWriter, r *http.Request) {
	hexId := bson.ObjectIdHex(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])

	c := session.DB("imagedb").C("images")

	var image Image
	err := c.FindId(hexId).One(&image)
	if err != nil {
		panic(err)
	}

	switch r.Method {
	case "GET":
		render(w, `
		{{define "title"}}Tags list{{end}}
		{{define "body"}}
		<form method="POST">
			<dl>
				<dt>Original name</dt>
				<dd>{{.OriginalName}}</dd>
				<dt>Tags</dt>
				<dd>
					<input type="text" name="tags" value="{{.TagsString}}" autofocus>
				</dd>
			</dl>
			<input type="submit" value="Save">
		</form>
		<img src="{{.RawLink}}">
		{{end}}
		`, image)

	case "POST":
		image.Tags = tagsFromString(r.FormValue("tags"))

		err := c.UpdateId(hexId, image)
		if err != nil {
			panic(err)
		}

		w.Header()["Location"] = []string{image.Link()}
		w.WriteHeader(http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleTagsList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(nil).All(&images)
		if err != nil {
			panic(err)
		}

		_tags := map[Tag]bool{}
		for _, image := range images {
			for _, tag := range image.Tags {
				_tags[Tag(tag)] = true
			}
		}

		tags := []Tag{}
		for tag := range _tags {
			tags = append(tags, tag)
		}

		sort.Sort(TagByName(tags))

		render(w, `
		{{define "title"}}Tags list{{end}}
		{{define "body"}}
		<ul>
			{{range .}}
				<li><a href="{{.Link}}">{{.}}</a></li>
			{{end}}
		</ul>
		{{end}}
		`, tags)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		query := parseQuery(r.FormValue("q"))

		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(query).All(&images)
		if err != nil {
			panic(err)
		}

		listImages(w, r, images)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleUntagged(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(bson.M{
			"tags": []string{},
		}).All(&images)
		if err != nil {
			panic(err)
		}

		listImages(w, r, images)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if r.URL.Path != "/" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		links := []string{
			"/all",
			"/tags",
			"/search",
			"/untagged",
			"/download",
			"/upload",
			"/duplicates",
		}

		render(w, `
		{{define "title"}}Index{{end}}
		{{define "body"}}
		<ul>
			{{range .}}
				<li><a href="{{.}}">{{.}}</a></li>
			{{end}}
		</ul>
		{{end}}
		`, links)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")

	switch r.Method {
	case "GET":
		render(w, `
		{{define "title"}}Download{{end}}
		{{define "body"}}
		<form method="POST">
			<label>
				URL
				<input type="text" name="url" value="{{.url}}">
			</label>
			<label>
				Tags
				<input type="text" name="tags" autofocus>
			</label>
			<input type="submit">
		</form>
		<img src="{{.url}}">
		{{end}}
		`, map[string]interface{}{
			"url": url,
		})

	case "POST":
		tags := tagsFromString(r.FormValue("tags"))

		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		storedImage := addImage(resp.Body, tags, url)

		w.Header()["Location"] = []string{storedImage.Link()}
		w.WriteHeader(http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		tags := tagsFromString(r.FormValue("tags"))

		err := r.ParseMultipartForm(int64(math.Pow(2, 29))) // 512MB
		if err != nil {
			panic(err)
		}

		for _, file := range r.MultipartForm.File["file"] {
			f, err := file.Open()
			if err != nil {
				panic(err)
			}
			defer f.Close()

			addImage(f, tags, file.Filename)
		}

		w.Header()["Location"] = []string{
			"/search?q=" + url.QueryEscape(strings.Join(tags, " ")),
		}
		w.WriteHeader(http.StatusFound)

	case "GET":
		render(w, `
		{{define "title"}}Upload{{end}}
		{{define "body"}}
		<form method="POST" enctype="multipart/form-data">
			<label>
				Files
				<input type="file" name="file" multiple accept="image/*">
			</label>
			<label>
				Tags
				<input type="text" name="tags">
			</label>
			<input type="submit">
		</form>
		{{end}}
		`, nil)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleDuplicates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(bson.M{
			"tags": bson.M{
				"$nin": []string{"_delete"},
			},
		}).All(&images)
		if err != nil {
			panic(err)
		}

		duplicates := map[string][]Image{}

		for _, image := range images {
			if _, ok := duplicates[image.Hash()]; !ok {
				duplicates[image.Hash()] = []Image{}
			}
			duplicates[image.Hash()] = append(duplicates[image.Hash()], image)
		}

		duplicateImages := []Image{}

		for _, dups := range duplicates {
			if len(dups) > 1 {
				duplicateImages = append(duplicateImages, dups...)
			}
		}

		listImages(w, r, duplicateImages)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
