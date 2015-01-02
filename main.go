package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Image struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	OriginalName string
	ContentType  string
	Image        []byte
	Tags         []string
}

func (i Image) Link() string {
	return "/image/" + i.ID.Hex()
}

func (i Image) RawLink() string {
	return "/_image/" + i.ID.Hex()
}

func (i Image) TagsString() string {
	return strings.Join(i.Tags, " ")
}

type Tag string

func (t Tag) String() string {
	return string(t)
}

func (t Tag) Link() string {
	return "/tags/" + string(t)
}

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

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/all", handleAll)
	http.HandleFunc("/_image/", handleRawImage)
	http.HandleFunc("/image/", handleImage)
	http.HandleFunc("/tags", handleTagsList)
	http.HandleFunc("/tags/", handleTags)
	http.HandleFunc("/untagged", handleUntagged)

	log.Fatalln(http.ListenAndServe(":3000", nil))
}

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

func handleAll(w http.ResponseWriter, r *http.Request) {
	c := session.DB("imagedb").C("images")

	var images []Image
	err := c.Find(nil).All(&images)
	if err != nil {
		panic(err)
	}

	listImages(w, images)
}

func handleRawImage(w http.ResponseWriter, r *http.Request) {
	hexId := bson.ObjectIdHex(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])

	c := session.DB("imagedb").C("images")

	var image Image
	err := c.FindId(hexId).One(&image)
	if err != nil {
		panic(err)
	}

	w.Header()["Content-type"] = []string{image.ContentType}
	w.Write(image.Image)
}

func handleImage(w http.ResponseWriter, r *http.Request) {
	hexId := bson.ObjectIdHex(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])

	c := session.DB("imagedb").C("images")

	var image Image
	err := c.FindId(hexId).One(&image)
	if err != nil {
		panic(err)
	}

	if r.Method == "POST" {
		image.Tags = strings.Split(r.FormValue("tags"), " ")

		err := c.UpdateId(hexId, image)
		if err != nil {
			panic(err)
		}

		w.Header()["Location"] = []string{image.Link()}
		w.WriteHeader(http.StatusFound)

		return
	}

	err = template.Must(template.New("").Parse(`<!doctype html>
	<form method="POST">
	<dl>
	<dt>Original name</dt>
	<dd>{{.OriginalName}}</dd>
	<dt>Tags</dt>
	<dd>
	<input type="text" name="tags" value="{{.TagsString}}">
	</dd>
	</dl>
	<input type="submit" value="Save">
	</form>
	<img src="{{.RawLink}}">
	`)).Execute(w, image)
	if err != nil {
		panic(err)
	}
}

func handleTagsList(w http.ResponseWriter, r *http.Request) {
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
}

func handleTags(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	c := session.DB("imagedb").C("images")

	var images []Image
	err := c.Find(bson.M{
		"tags": tag,
	}).All(&images)
	if err != nil {
		panic(err)
	}

	listImages(w, images)
}

func handleUntagged(w http.ResponseWriter, r *http.Request) {
	c := session.DB("imagedb").C("images")

	var images []Image
	err := c.Find(bson.M{
		"tags": []string{},
	}).All(&images)
	if err != nil {
		panic(err)
	}

	listImages(w, images)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	links := []string{
		"/all",
		"/tags",
		"/untagged",
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
}
