package main

import (
	"encoding/json"
	"io/ioutil"
	"sort"
)

type TagPostMap map[string][]string
type TagPopularity []TagCount

type Tags struct {
	TagsFile string `json:"-"`
	// Collection of Post objects, indexed by file path.
	Post map[string]*Post
	// For each tag found, a collection of file paths.
	Tag TagPostMap
}

type TagCount struct {
	Tag   string
	Count int
}

func NewTags(tagsFile string, postPath string) *Tags {
	tags := &Tags{tagsFile, make(map[string]*Post), make(map[string][]string)}
	// Ignore errors here. It's ok if we can't load, which usually just means that the
	// tags file doesn't exist.
	err := tags.Load()
	if err != nil {
		err = tags.Generate(postPath)
		if err != nil {
			logger.Println("NewTags:", err)
			return tags
		}
		tags.Save()
	}
	return tags
}

func (tags *Tags) AddPost(post *Post) {
	tags.Post[post.SourcePath] = post
	for _, tag := range post.Tags {
		tags.AddPostTag(post, tag)
	}
}

func (tags *Tags) AddPostTag(post *Post, tag string) {
	l := tags.Tag[tag]
	if l == nil {
		l = []string{}
	}
	l = append(l, post.SourcePath)
	tags.Tag[tag] = l
}

func (tags *Tags) Load() error {
	buf, err := ioutil.ReadFile(tags.TagsFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, tags)
	if err != nil {
		return err
	}
	return nil
}

func (tags *Tags) Save() error {
	buf, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tags.TagsFile, buf, 0600)
}

func (tags *Tags) Generate(postPath string) error {
	// Walk through postPath, finding all posts.
	// On each file that successfully parses, add it to the map.

	debug("Generating tags file")

	err := tags.readPosts(postPath)
	if err != nil {
		return err
	}

	// Now we have the Post headers. Generate the tag collections from that.
	for _, post := range tags.Post {
		for _, tag := range post.Tags {
			tags.AddPostTag(post, tag)
		}
	}

	return nil
}

func (tags *Tags) readPosts(postPath string) error {
	postList, err := LoadPostsFromPath(postPath, true)
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

func (tags *Tags) TagsByPopularity() TagPopularity {
	tp := make(TagPopularity, len(tags.Tag))
	i := 0
	for tag, posts := range tags.Tag {
		tp[i] = TagCount{tag, len(posts)}
		i++
	}

	sort.Sort(tp)

	return tp
}

func (pop TagPopularity) Less(i, j int) bool {
	// TODO Sort by count or by title?
	// Reverse sort by default.
	return pop[i].Count > pop[j].Count
}

func (pop TagPopularity) Len() int {
	return len(pop)
}

func (pop TagPopularity) Swap(i, j int) {
	pop[i], pop[j] = pop[j], pop[i]
}
