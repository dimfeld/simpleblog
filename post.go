package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/russross/blackfriday"
	"os"
	"path"
	"sort"
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
	dir, err := os.Open(postPath)
	if err != nil {
		return nil, err
	}

	files, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	postList := make(PostList, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		newPost, err := NewPost(path.Join(postPath, file.Name()), readContent)
		if err == nil {
			postList = append(postList, newPost)
		}
		// else some logging?
	}

	return postList, nil
}

func PreviousMonthDir(postBase string, current time.Time) (time.Time, error) {
	// Get the post directory
	postDir, err := os.Open(postBase)
	if err != nil {
		return time.Time{}, err
	}
	postDirStat, err := postDir.Stat()
	if err != nil || !postDirStat.IsDir() {
		return time.Time{}, errors.New("Post path is not directory")
	}

	// Read out the list of year directories.
	yearDirs, err := postDir.Readdir(0)
	if err != nil {
		return time.Time{}, err
	}

	startMonth := current.Month() - 1
	year := current.Year()

	// Convert them all to ints.
	yearInts := make(sort.IntSlice, 0, len(yearDirs))
	for _, y := range yearDirs {
		if !y.IsDir() {
			continue
		}

		yearInt, err := strconv.Atoi(y.Name())
		if err != nil {
			continue
		}

		if yearInt <= year {
			// Add all years that are less than or equal to the current one.
			yearInts = append(yearInts, yearInt)
		}
	}
	// Reverse sort so we start with the most recent year.
	sort.Sort(sort.Reverse(yearInts))

	for _, year = range yearInts {
		for month := startMonth; month > 0; month-- {
			monthPath := PostPath(postBase, year, month)
			stat, err := os.Stat(monthPath)
			if err == nil && stat.IsDir() {
				return time.Date(year, month, 1, 1, 1, 1, 1, time.UTC), nil
			}
		}
		// Going into the previous year, always start with December.
		startMonth = time.December
	}

	return time.Time{}, os.ErrNotExist
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
