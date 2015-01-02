package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"

	"gopkg.in/mgo.v2"
)

var (
	session *mgo.Session
	noop    bool
	tags    _tags
)

func main() {
	var err error
	session, err = mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	flag.BoolVar(&noop, "n", false, "do nothing")
	flag.Var(&tags, "t", "tags (more than one allowed)")
	flag.Parse()

	displayCount()

	if len(flag.Args()) == 0 {
		fmt.Println("Usage: import [-n] [-t tag] [-t tag2] img1.jpg img2.jpg [...]")
		os.Exit(1)
	}

	for _, path := range flag.Args() {
		addImage(path, tags)
	}

	displayCount()
}

func displayCount() {
	c := session.DB("imagedb").C("images")
	count, err := c.Count()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Number of images: %d\n", count)
}

func addImage(path string, tags []string) {
	c := session.DB("imagedb").C("images")

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	image := make([]byte, int(math.Pow(2, 22))+1)
	n, err := io.ReadFull(f, image)
	if err != nil && err != io.ErrUnexpectedEOF {
		panic(err)
	}
	if n > int(math.Pow(2, 22)) {
		// the image is bigger than 4MB!
		panic(fmt.Errorf(`image "%s" too big: %d bytes`, path, n))
	}

	mimeType := http.DetectContentType(image)

	fmt.Printf("Adding image: %s, %s, %s\n", path, mimeType, tags)

	if !noop {
		err = c.Insert(&Image{
			OriginalName: path,
			ContentType:  mimeType,
			Image:        image[:n],
			Tags:         tags,
		})
		if err != nil {
			panic(err)
		}
	}
}

type Image struct {
	OriginalName string
	ContentType  string
	Image        []byte
	Tags         []string
}

type _tags []string

func (t *_tags) Set(value string) error {
	*t = append(*t, value)
	return nil
}

func (t *_tags) String() string {
	return fmt.Sprintf("%#v", t)
}
