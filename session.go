package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Session struct {
	Host  string
	mongo *mgo.Session
}

func (s *Session) Connect() error {
	session, err := mgo.Dial(s.Host)
	if err != nil {
		return err
	}
	s.mongo = session
	return nil
}

func (s *Session) Close() {
	s.mongo.Close()
}

func (s *Session) Find(query bson.M) ([]Image, error) {
	var images []Image
	c := s.mongo.DB("imagedb").C("images")

	err := c.Find(query).All(&images)
	return images, err
}

func (s *Session) FindId(hexId string) (Image, error) {
	var image Image
	c := s.mongo.DB("imagedb").C("images")

	err := c.FindId(bson.ObjectIdHex(hexId)).One(&image)
	return image, err
}

func (s *Session) OpenRawImage(hexId string) (*mgo.GridFile, error) {
	gridfs := s.mongo.DB("imagedb").GridFS("raw_images2")

	return gridfs.OpenId(bson.ObjectIdHex(hexId))
}

func (s *Session) UpdateId(hexId string, image Image) error {
	c := s.mongo.DB("imagedb").C("images")
	return c.UpdateId(hexId, image)
}

func (s *Session) CreateRawImage() (*mgo.GridFile, error) {
	gridfs := s.mongo.DB("imagedb").GridFS("raw_images2")

	return gridfs.Create("")
}

func (s *Session) Insert(image Image) error {
	c := s.mongo.DB("imagedb").C("images")

	return c.Insert(image)
}
