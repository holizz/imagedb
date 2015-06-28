package db

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

func (i *Image) SetTags(tags []string) {
	tagMap := map[string]bool{}
	for _, t := range tags {
		tagMap[t] = true
	}
	newTags := []string{}
	for t, _ := range tagMap {
		newTags = append(newTags, t)
	}
	i.Tags = newTags
}

func (i Image) TagsString() string {
	return strings.Join(i.Tags, " ")
}

func (i Image) Hash(session *Session) (string, error) {
	if i.hash == "" {
		image, err := session.OpenRawImage(i.RawImage)
		if err != nil {
			return "", fmt.Errorf("(Image) Hash: could not find raw image for %s %v", i.Link(), err)
		}
		defer image.Close()

		hash := &xxhash.XXHash{}

		_, err = io.Copy(hash, image)
		if err != nil {
			return "", fmt.Errorf("(Image) Hash: failed copying: %v", err)
		}
		i.hash = fmt.Sprintf("%08x", hash.Sum32())

		i.Update(session)
	}

	return i.hash, nil
}

func (i *Image) Update(session *Session) {
	session.UpdateId(i.ID.Hex(), *i)
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
