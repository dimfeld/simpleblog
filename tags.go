package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (tags *Tags) Generate(postPath string) error {
	// Walk through postPath, finding all posts.
	// On each file that successfully parses, add it to the map.

	err := tags.readPostHeaders(postPath)
	if err != nil {
		return err
	}

	// Now we have the Post headers. Generate the tag collections from that.
	for _, post := range tags.PostHeader {
		for _, tag := range post.Tags {
			l := tags.Tag[tag]
			if l == nil {
				l = make([]string, 0)
			}
			l = append(l, post.SourcePath)
			tags.Tag[tag] = l
		}
	}

	return nil
}

func (tags *Tags) postHeadersWalkFunc(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		// Nothing to do for directories
		return nil
	}

	post, err := NewPostHeader(path)
	if err != nil {
		// Returning an error here will stop the walk, so log the error some other way.
		return nil
	}

	tags.PostHeader[path] = post
	return nil
}

func (tags *Tags) readPostHeaders(postPath string) error {
	return filepath.Walk(postPath, tags.postHeadersWalkFunc)
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

// Less compares two PostHeader objects in a PostList by date.
func (l PostList) Less(i, j int) bool {
	return l[i].Timestamp.Before(l[j].Timestamp)
}

func (l PostList) Len() int {
	return len(l)
}

func (l PostList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
