package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

type Post struct {
	SourcePath string
	Title      string
	UrlTitle   string
	Timestamp  time.Time
	Tags       []string
	Content    []byte
}

func urlTitleMap(r rune) rune {
	if !unicode.IsPrint(r) || unicode.IsPunct(r) {
		return -1
	}

	if unicode.IsSpace(r) {
		return '-'
	}

	return unicode.ToLower(r)
}

func MakeUrlTitle(s string) string {
	return strings.Map(urlTitleMap, s)
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

	p = &Post{SourcePath: filePath}

	reader := bufio.NewReader(f)

	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	p.Title = string(line[0 : len(line)-1])
	p.UrlTitle = MakeUrlTitle(p.Title)

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
		return p, fmt.Errorf("Unexpected input %s", string(line))
	}

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
