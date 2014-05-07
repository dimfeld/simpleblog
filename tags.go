package main

import (
	"encoding/json"
	"io/ioutil"
)

type Tags struct {
	// Collection of PostHeader objects, indexed by file path.
	PostHeader map[string]*PostHeader
	// For each tag found, a collection of file paths.
	Tag map[string][]string
}

func NewTags() *Tags {
	return &Tags{make(map[string]*PostHeader), make(map[string][]string)}
}

func LoadTags(tagFile string) (*Tags, error) {
	buf, err := ioutil.ReadFile(tagFile)
	if err != nil {
		return nil, err
	}
	tags := &Tags{}
	err = json.Unmarshal(buf, tags.Tag)
	return tags, err
}

func (tags *Tags) Save(tagFile string) error {
	buf, err := json.Marshal(tags.Tag)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tagFile, buf, 0600)
}

func (tags *Tags) Generate(postPath string) (*Tags, error) {
	// Walk through postPath, finding all posts.
	// On each file that successfully parses, add it to the map.
	return nil, nil
}

func (tags *Tags) readPostHeaders(postPath string) {

}
