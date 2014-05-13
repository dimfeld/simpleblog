package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/russross/blackfriday"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	p.Title = strings.TrimSpace(string(line[0 : len(line)-1]))

	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}

	p.Timestamp, err = time.Parse("1/2/06 3:04PM MST", strings.TrimSpace(line[0:len(line)-1]))
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

	err = p.readHeader(reader)
	if err != nil {
		return
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

func (p *Post) HTMLContent() []byte {
	return blackfriday.MarkdownCommon(p.Content)
}

func LoadPostsFromPath(postPath string, readContent bool) (PostList, error) {
	var outerErr error = nil
	postList := make(PostList, 0, 15)
	err := filepath.Walk(postPath,
		func(filePath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			newPost, err := NewPost(filePath, readContent)
			if err == nil {
				postList = append(postList, newPost)
			} else {
				logger.Printf("Failed parsing post at %s: %s", filePath, err)
				if outerErr == nil {
					// Pass the error outward.
					outerErr = err
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	return postList, outerErr
}

func NewArchiveSpecList(postBase string) (ArchiveSpecList, error) {
	// Get the post directory.
	postDir, err := os.Open(postBase)
	if err != nil {
		return nil, err
	}
	postDirStat, err := postDir.Stat()
	if err != nil || !postDirStat.IsDir() {
		return nil, errors.New("Post path is not directory")
	}

	// Read out the list of year directories.
	yearDirs, err := postDir.Readdir(0)
	if err != nil {
		return nil, err
	}

	list := make(ArchiveSpecList, 0)

	for _, yearDirStat := range yearDirs {
		if !yearDirStat.IsDir() {
			continue
		}

		yearDirName := yearDirStat.Name()
		yearInt, err := strconv.Atoi(yearDirName)
		if err != nil {
			// This isn't a numeric path. Ignore it.
			continue
		}

		yearDirPath := path.Join(postBase, yearDirName)
		yearDir, err := os.Open(yearDirPath)
		if err != nil {
			// Probably the directory was deleted. Log and move on.
			logger.Println("NewArchiveSpecList: Failed to open", yearDirPath)
			continue
		}

		monthDirs, err := yearDir.Readdir(0)
		if err != nil {
			logger.Println("NewArchiveSpecList: Failed to read files from", yearDirPath)
		}

		for _, monthDirSpec := range monthDirs {
			if !monthDirSpec.IsDir() {
				continue
			}
			monthInt, err := strconv.Atoi(monthDirSpec.Name())
			if err != nil {
				// This isn't a numeric path. Ignore it.
				continue
			}

			spec := time.Date(yearInt, time.Month(monthInt), 1, 1, 1, 1, 1, time.UTC)
			list = append(list, ArchiveSpec(spec))
		}
	}
	return list, nil
}

func PostPath(base string, year int, month time.Month) string {
	return path.Join(base, strconv.Itoa(year), fmt.Sprintf("%02d", int(month)))
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
