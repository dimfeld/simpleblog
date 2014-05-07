package main

import (
	"encoding/json"
	"io/ioutil"
	"sort"
)

type PostList []*PostHeader
type PostMap map[string][]string

type Tags struct {
	// Collection of PostHeader objects, indexed by file path.
	PostHeader map[string]*PostHeader
	// For each tag found, a collection of file paths.
	Tag PostMap
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

func (tags *Tags) PostsByDate(tag string) PostList {
	paths := tags.Tag[tag]
	l := make(PostList, len(paths))
	for i, path := range paths {
		l[i] = tags.PostHeader[path]
	}
	sort.Sort(l)
	return l
}

func (tags *Tags) TagsByName() []string {
	s := make([]string, len(tags.Tag))
	i := 0
	for tag, _ := range tags.Tag {
		s[i] = tag
	}

	sort.Strings(s)
	return s
}

func (tags *Tags) TagsByPopularity() []string {
	s := make([]string, len(tags.Tag))
	i := 0
	for tag, _ := range tags.Tag {
		s[i] = tag
	}

	// TODO Need to do the sort here.

	return s
}

func (l PostList) Less(i, j int) bool {
	return l[i].Timestamp.Before(l[j].Timestamp)
}

func (l PostList) Len() int {
	return len(l)
}

func (l PostList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
