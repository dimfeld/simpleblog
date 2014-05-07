package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/russross/blackfriday"
	"os"
	"strings"
	"time"
)

type PostHeader struct {
	SourcePath string
	Title      string
	Timestamp  time.Time
	Tags       []string
}

type Post struct {
	PostHeader
	Content []byte
}

func (p *PostHeader) readHeader(reader *bufio.Reader) (err error) {
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

// NewPostHeader works similarly to NewPost, except it only reads and returns a header
func NewPostHeader(filePath string) (p *PostHeader, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	p = &PostHeader{SourcePath: filePath}
	reader := bufio.NewReader(f)
	err = p.readHeader(reader)
	f.Close()
	return
}

// NewPost reads a post from disk and returns a Post containing its data.
// If readContent is false, only the header of the post is read.
// The post format is:
// Title
// Date/Time
// Tags
//
// Markdown Content
func NewPost(filePath string) (p *Post, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p = &Post{}
	p.SourcePath = filePath

	reader := bufio.NewReader(f)

	p.PostHeader.readHeader(reader)

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return
	}
	p.Content = buf.Bytes()

	return
}

func (p *Post) HTML() []byte {
	return blackfriday.MarkdownCommon(p.Content)
}
