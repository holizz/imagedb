package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/vova616/xxhash"
	"gopkg.in/mgo.v2/bson"
)

type Image struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	OriginalName string
	Tags         []string
	RawImage     string
	hash         string
}

func (i Image) Link() string {
	return "/image/" + i.ID.Hex()
}

func (i Image) RawLink() string {
	return "/_image/" + i.RawImage
}

func (i Image) TagsString() string {
	return strings.Join(i.Tags, " ")
}

func (i Image) Hash() string {
	if i.hash == "" {
		image, err := session.OpenRawImage(i.RawImage)
		if err != nil {
			panic(fmt.Errorf("could not find raw image for %s %#v", i.Link(), err))
		}
		defer image.Close()

		hash := &xxhash.XXHash{}

		_, err = io.Copy(hash, image)
		if err != nil {
			panic(err)
		}
		i.hash = fmt.Sprintf("%08x", hash.Sum32())

		session.UpdateId(i.ID.String(), i)
	}

	return i.hash
}

type Tag string

func (t Tag) String() string {
	return string(t)
}

func (t Tag) Link() string {
	return "/search?q=" + url.QueryEscape(string(t))
}

type TagByName []Tag

func (a TagByName) Len() int           { return len(a) }
func (a TagByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TagByName) Less(i, j int) bool { return a[i] < a[j] }
