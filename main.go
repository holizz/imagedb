package main

import (
	"html/template"
	"log"
	"net/http"

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

func (i *Image) Link() string {
	return "/_image/" + i.ID.Hex()
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

	http.HandleFunc("/all", handleAll)

	log.Fatalln(http.ListenAndServe(":3000", nil))
}

func handleAll(w http.ResponseWriter, r *http.Request) {
	c := session.DB("imagedb").C("images")

	var images []Image
	err := c.Find(nil).All(&images)
	if err != nil {
		panic(err)
	}

	template.Must(template.New("").Parse(`<!doctype html>
	<ul>
	{{range .}}
	<li><a href="{{.Link}}">{{.Link}}</a></li>
	{{end}}
	</ul>
	`)).Execute(w, images)
}
