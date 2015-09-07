package db

import (
	"strings"

	"gopkg.in/mgo.v2/bson"
)

func parseQuery(s string) bson.M {
	if s == ":all" {
		return bson.M{}
	}

	if s == ":untagged" {
		return bson.M{
			"tags": []string{},
		}
	}

	tags := TagsFromString(s)
	return bson.M{
		"tags": bson.M{
			"$all": tags,
		},
	}
}

func TagsFromString(s string) []string {
	tags := []string{}
	for _, t := range strings.Split(s, " ") {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
