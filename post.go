package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/russross/blackfriday"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PostList []*Post

type Post struct {
	SourcePath string
	Title      string
	Timestamp  time.Time
	Tags       []string
	Content    []byte
}

func (p *Post) readHeader(reader *bufio.Reader) (err error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	p.Title = string(line[0 : len(line)-1])

	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}

	p.Timestamp, err = time.Parse(time.RFC822, line[0:len(line)-1])
	if err != nil {
		return
	}

	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}
	p.Tags = strings.Split(line, ",")
	for i := range p.Tags {
		p.Tags[i] = strings.Title(strings.TrimSpace(p.Tags[i]))
	}

	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}
	if len(line) != 1 {
		return fmt.Errorf("Unexpected input after header: %s", string(line))
	}

	return nil
}

// NewPost reads a post from disk and returns a Post containing its data.
// If readContent is false, only the header of the post is read.
// The post format is:
// Title
// Date/Time
// Tags
//
// Markdown Content
func NewPost(filePath string, readContent bool) (p *Post, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p = &Post{}
	p.SourcePath = filePath

	reader := bufio.NewReader(f)

	p.readHeader(reader)

	if readContent {
		buf := &bytes.Buffer{}
		_, err = buf.ReadFrom(reader)
		if err != nil {
			return
		}
		p.Content = buf.Bytes()
	}

	return
}

func (p *Post) HTMLContent() []byte {
	return blackfriday.MarkdownCommon(p.Content)
}

func LoadPostsFromPath(postPath string, readContent bool) (PostList, error) {
	postList := make(PostList, 1)
	err := filepath.Walk(postPath,
		func(filePath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			newPost, err := NewPost(filePath, readContent)
			if err != nil {
				postList = append(postList, newPost)
			}
			return nil
		})
	return postList, err
}

// Less compares two PostH objects in a PostList by date.
func (l PostList) Less(i, j int) bool {
	return l[i].Timestamp.Before(l[j].Timestamp)
}

func (l PostList) Len() int {
	return len(l)
}

func (l PostList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
