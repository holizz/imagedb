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

	// Main handlers
	http.Handle("/", common.ThenFunc(handleRoot(session)))
	http.Handle("/all", common.ThenFunc(handleAll(session)))
	http.Handle("/_image/", common.ThenFunc(handleRawImage(session)))
	http.Handle("/image/", common.ThenFunc(handleImage(session)))
	http.Handle("/tags", common.ThenFunc(handleTagsList(session)))
	http.Handle("/untagged", common.ThenFunc(handleUntagged(session)))
	http.Handle("/download", common.ThenFunc(handleDownload(session)))
	http.Handle("/upload", common.ThenFunc(handleUpload(session)))
	http.Handle("/search", common.ThenFunc(handleSearch(session)))
	http.Handle("/rename", common.ThenFunc(handleRename(session)))

	log.Fatalln(http.ListenAndServe(":"+port, nil))
}

func handleAll(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listImages(w, r, session, ":all")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
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
			image.SetTags(tagsFromString(r.FormValue("tags")))

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

func handleTagsList(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			images, err := session.Find(nil)
			if err != nil {
				panic(err)
			}

			_tags := map[db.Tag]bool{}
			for _, image := range images {
				for _, tag := range image.Tags {
					_tags[db.Tag(tag)] = true
				}
			}

			tags := []db.Tag{}
			for tag := range _tags {
				tags = append(tags, tag)
			}

			sort.Sort(db.TagByName(tags))

			render(w, `
		{{define "title"}}Tags list{{end}}
		{{define "body"}}
		<table>
			<tbody>
				<tr>
					<th>Tag</th>
					<th>Actions</th>
				</tr>
				{{range .}}
					<tr>
						<td><a href="{{.Link}}">{{.}}</a></td>
						<td>
							<a href="/rename?from={{.}}">Rename</a>
						</td>
					</tr>
				{{end}}
			</tbody>
		</table>
		{{end}}
		`, tags)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleSearch(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listImages(w, r, session, r.FormValue("q"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleUntagged(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listImages(w, r, session, ":untagged")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleRoot(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
				"/rename",
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
}

func handleDownload(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

			storedImage := addImage(session, resp.Body, tags, url)

			w.Header()["Location"] = []string{storedImage.Link()}
			w.WriteHeader(http.StatusFound)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleUpload(session *db.Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
			query := parseQuery(from)

			images, err := session.Find(query)
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
			images, err := session.Find(nil)
			if err != nil {
				panic(err)
			}

			_tags := map[db.Tag]bool{}
			for _, image := range images {
				for _, tag := range image.Tags {
					_tags[db.Tag(tag)] = true
				}
			}

			tags := []db.Tag{}
			for tag := range _tags {
				tags = append(tags, tag)
			}

			sort.Sort(db.TagByName(tags))

			renderJson(w, struct {
				Num     int64
				Results []db.Tag
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
			query := parseQuery(r.FormValue("q"))

			images, err := session.Find(query)
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
