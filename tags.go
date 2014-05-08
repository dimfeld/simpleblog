package main

import (
	"encoding/json"
	"io/ioutil"
	"sort"
)

type TagPostMap map[string][]string

type Tags struct {
	// Collection of Post objects, indexed by file path.
	Post map[string]*Post
	// For each tag found, a collection of file paths.
	Tag TagPostMap
}

func NewTags() *Tags {
	return &Tags{make(map[string]*Post), make(map[string][]string)}
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
	for _, post := range tags.Post {
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

func (tags *Tags) readPostHeaders(postPath string) error {
	postList, err := LoadPostsFromPath(postPath, false)
	if err != nil {
		return err
	}

	for _, post := range postList {
		tags.Post[post.SourcePath] = post
	}
	return nil
}

func (tags *Tags) PostsByDate(tag string) PostList {
	paths := tags.Tag[tag]
	l := make(PostList, len(paths))
	for i, path := range paths {
		l[i] = tags.Post[path]
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
