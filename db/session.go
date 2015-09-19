package db

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Session struct {
	Host  string
	mongo *mgo.Session
}

func NewSessionFromEnv() *Session {
	mongoHost := os.Getenv("MONGODB")
	if len(mongoHost) == 0 {
		mongoHost = "localhost:27017"
	}

	return &Session{Host: mongoHost}
}

func (s *Session) Connect() error {
	session, err := mgo.Dial(s.Host)
	if err != nil {
		return fmt.Errorf("(*Session) Connect: %v", err)
	}
	s.mongo = session
	return nil
}

func (s *Session) Close() {
	s.mongo.Close()
}

func (s *Session) Find(query string) ([]Image, error) {
	var images []Image
	c := s.mongo.DB("imagedb").C("images")

	err := c.Find(parseQuery(query)).All(&images)
	if err != nil {
		return nil, fmt.Errorf("(*Session) Find: %v", err)
	}
	return images, nil
}

func (s *Session) FindId(hexId string) (Image, error) {
	var image Image
	c := s.mongo.DB("imagedb").C("images")

	err := c.FindId(bson.ObjectIdHex(hexId)).One(&image)
	if err != nil {
		return Image{}, fmt.Errorf("(*Session) FindId: %v", err)
	}
	return image, nil
}

func (s *Session) OpenRawImage(hexId string) (*mgo.GridFile, error) {
	gridfs := s.mongo.DB("imagedb").GridFS("raw_images2")

	file, err := gridfs.OpenId(bson.ObjectIdHex(hexId))
	if err != nil {
		return nil, fmt.Errorf("(*Session) OpenRawImage: %v", err)
	}
	return file, nil
}

func (s *Session) UpdateId(hexId string, image Image) error {
	c := s.mongo.DB("imagedb").C("images")
	err := c.UpdateId(bson.ObjectIdHex(hexId), image)
	if err != nil {
		return fmt.Errorf("(*Session) UpdateId: %v", err)
	}
	return nil
}

func (s *Session) CreateRawImage() (*mgo.GridFile, error) {
	gridfs := s.mongo.DB("imagedb").GridFS("raw_images2")

	file, err := gridfs.Create("")
	if err != nil {
		return nil, fmt.Errorf("(*Session) CreateRawImage: %v", err)
	}
	return file, nil
}

func (s *Session) Insert(image Image) error {
	c := s.mongo.DB("imagedb").C("images")

	err := c.Insert(image)
	if err != nil {
		return fmt.Errorf("(*Session) Insert: %v", err)
	}
	return nil
}

func (s *Session) Tags() ([]TagInfo, error) {
	images, err := s.Find(":all")
	if err != nil {
		return nil, err
	}

	tags := map[Tag]int64{}
	for _, image := range images {
		for _, tag := range image.Tags {
			tags[Tag(tag)] += 1
		}
	}

	tagsWithInfo := []TagInfo{}

	for tag, num := range tags {
		tagsWithInfo = append(tagsWithInfo, TagInfo{
			Name: string(tag),
			Num:  num,
		})
	}

	sort.Sort(TagInfoByName(tagsWithInfo))

	return tagsWithInfo, nil
}
