package main

import (
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/holizz/imagedb/db"
	"github.com/justinas/alice"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	session := db.NewSessionFromEnv()

	err := session.Connect()
	if err != nil {
		panic(err)
	}
	defer session.Close()

	common := alice.New(handleLogging)

	// Static
	http.Handle("/assets/", common.Then(http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))))
	http.Handle("/bower_components/", common.Then(http.StripPrefix("/bower_components/", http.FileServer(http.Dir("bower_components")))))

	// API
	http.Handle("/api/tags", common.ThenFunc(handleApiTags(session)))
	http.Handle("/api/search", common.ThenFunc(handleApiSearch(session)))

	// Serve polymer app
	http.Handle("/", common.ThenFunc(handleRoot(session)))

	// Serve images
	http.Handle("/_image/", common.ThenFunc(handleRawImage(session)))

	// Soon-to-be-legacy handlers
	http.Handle("/image/", common.ThenFunc(handleImage(session)))
	http.Handle("/download", common.ThenFunc(handleDownload(session)))
	http.Handle("/upload", common.ThenFunc(handleUpload(session)))
	http.Handle("/rename", common.ThenFunc(handleRename(session)))

	log.Fatalln(http.ListenAndServe(":"+port, nil))
}

func handleRawImage(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			hexId := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

			image, err := session.OpenRawImage(hexId)
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
}

func handleImage(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hexId := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

		image, err := session.FindId(hexId)
		if err != nil {
			panic(err)
		}

		switch r.Method {
		case "GET":
			render(w, `
			{{define "title"}}Image{{end}}
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
			image.SetTags(db.TagsFromString(r.FormValue("tags")))

			err := session.UpdateId(hexId, image)
			if err != nil {
				panic(err)
			}

			w.Header()["Location"] = []string{image.Link()}
			w.WriteHeader(http.StatusFound)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleRoot(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			render(w, `
			{{define "title"}}Index{{end}}
			{{define "body"}}
			<image-db></image-db>
			{{end}}
			`, nil)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleDownload(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		switch r.Method {
		case "GET":
			urls := r.Form["url"]
			render(w, `
			{{define "title"}}Download{{end}}
			{{define "body"}}
				<form method="POST">
					<label>
						URL
						{{range .url}}
							<input type="text" name="url" value="{{.}}">
						{{end}}
					</label>
					<label>
						Tags
						<input type="text" name="tags" autofocus>
					</label>
					<input type="submit">
				</form>
				{{range .url}}
					<img src="{{.}}">
				{{end}}
			{{end}}
			`, map[string]interface{}{
				"url": urls,
			})

		case "POST":
			urls := r.PostForm["url"]
			tags := db.TagsFromString(r.FormValue("tags"))

			var storedImage db.Image

			for _, url := range urls {
				resp, err := http.Get(url)
				if err != nil {
					panic(err)
				}

				storedImage = addImage(session, resp.Body, tags, url)
			}

			if len(urls) == 1 {
				w.Header()["Location"] = []string{storedImage.Link()}
				w.WriteHeader(http.StatusFound)
			} else {
				w.Header()["Location"] = []string{
					"/search?q=" + url.QueryEscape(strings.Join(tags, " ")),
				}
				w.WriteHeader(http.StatusFound)
			}

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleUpload(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			tags := db.TagsFromString(r.FormValue("tags"))

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

				addImage(session, f, tags, file.Filename)
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
}

func handleRename(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.FormValue("from")
		to := r.FormValue("to")

		switch r.Method {
		case "GET":
			render(w, `
			{{define "title"}}Rename{{end}}
			{{define "body"}}
			<form method="POST">
				<label>
					From
					<input type="text" name="from" value="{{.from}}">
				</label>
				<label>
					To
					<input type="text" name="to" value="{{.to}}">
				</label>
				<input type="submit">
			</form>
			{{end}}
			`, map[string]interface{}{
				"from": from,
				"to":   to,
			})

		case "POST":
			images, err := session.Find(from)
			if err != nil {
				panic(err)
			}

			for _, image := range images {
				tags := []string{}

				for _, tag := range image.Tags {
					if tag == from {
						tags = append(tags, to)
					} else {
						tags = append(tags, tag)
					}
				}

				image.SetTags(tags)

				image.Update(session)
			}

			w.Header()["Location"] = []string{
				"/search?q=" + url.QueryEscape(to),
			}
			w.WriteHeader(http.StatusFound)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleApiTags(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			tags, err := session.Tags()
			if err != nil {
				panic(err)
			}

			renderJson(w, struct {
				Num     int64
				Results []db.TagInfo
			}{
				Num:     int64(len(tags)),
				Results: tags,
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleApiSearch(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			images, err := session.Find(r.FormValue("q"))
			if err != nil {
				panic(err)
			}

			renderJson(w, struct {
				Num     int64
				Results []db.Image
			}{
				Num:     int64(len(images)),
				Results: images,
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
