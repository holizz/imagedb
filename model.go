package main

import (
	"strings"

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

type TagByName []Tag

func (a TagByName) Len() int           { return len(a) }
func (a TagByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TagByName) Less(i, j int) bool { return a[i] < a[j] }
