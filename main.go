package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
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
	session, err = mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	common := alice.New(handleLogging)

	http.Handle("/", common.ThenFunc(handleRoot))
	http.Handle("/all", common.ThenFunc(handleAll))
	http.Handle("/_image/", common.ThenFunc(handleRawImage))
	http.Handle("/image/", common.ThenFunc(handleImage))
	http.Handle("/tags", common.ThenFunc(handleTagsList))
	http.Handle("/untagged", common.ThenFunc(handleUntagged))
	http.Handle("/download", common.ThenFunc(handleDownload))
	http.Handle("/upload", common.ThenFunc(handleUpload))
	http.Handle("/search", common.ThenFunc(handleSearch))

	log.Fatalln(http.ListenAndServe(":3000", nil))
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

		c := session.DB("imagedb").C("raw_images")

		var image RawImage
		err := c.FindId(hexId).One(&image)
		if err != nil {
			panic(err)
		}

		w.Header()["Content-type"] = []string{image.ContentType}
		w.Write(image.Image)
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
		err = template.Must(template.New("").Parse(`<!doctype html>
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
		`)).Execute(w, image)
		if err != nil {
			panic(err)
		}

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

		err = template.Must(template.New("").Parse(`<!doctype html>
		<ul>
		{{range .}}
		<li><a href="{{.Link}}">{{.}}</a></li>
		{{end}}
		</ul>
		`)).Execute(w, tags)
		if err != nil {
			panic(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		tags := tagsFromString(r.FormValue("q"))

		c := session.DB("imagedb").C("images")

		var images []Image
		err := c.Find(bson.M{
			"tags": bson.M{
				"$all": tags,
			},
		}).All(&images)
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
		}

		err := template.Must(template.New("").Parse(`<!doctype html>
		<ul>
		{{range .}}
		<li><a href="{{.}}">{{.}}</a></li>
		{{end}}
		</ul>
		`)).Execute(w, links)
		if err != nil {
			panic(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")

	switch r.Method {
	case "GET":
		err := template.Must(template.New("").Parse(`<!doctype html>
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
		`)).Execute(w, map[string]interface{}{
			"url": url,
		})
		if err != nil {
			panic(err)
		}

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
		err := template.Must(template.New("").Parse(`<!doctype html>
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
		`)).Execute(w, nil)
		if err != nil {
			panic(err)
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
