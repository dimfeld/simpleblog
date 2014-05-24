package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/dimfeld/blackfriday"
	"github.com/dimfeld/glog"
	"hash/fnv"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const PostTimeFormat string = "1/2/06 3:04PM -0700"

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

	p.Timestamp, err = time.Parse(PostTimeFormat, strings.TrimSpace(line[0:len(line)-1]))
	if err != nil {
		return
	}

	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}

	line = strings.TrimSpace(line)
	if line == "" {
		p.Tags = []string{}
	} else {
		p.Tags = strings.Split(line, ",")
	}
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
		glog.Errorf("Error reading post %s: %s", filePath, err.Error())
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

func (p *Post) HTMLContent(atom bool) template.HTML {
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_FOOTNOTE_RETURN_LINKS

	if atom {
		htmlFlags |= blackfriday.HTML_ABSOLUTE_LINKS
	} else {
		htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
		htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
		htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

	}

	domain := "http://" + config.Domain
	if domain[len(domain)-1] == '/' {
		domain = domain[0 : len(domain)-1]
	}

	// Take the hash of the path, to form a prefix for the footnote links.
	// This prevents duplicate anchors when multiple posts with footnotes are in a page.
	hash := fnv.New32a()
	hash.Write([]byte(p.SourcePath))
	prefix := strconv.FormatInt(int64(hash.Sum32()), 36)

	parameters := blackfriday.HtmlRendererParameters{
		AbsolutePrefix:             domain,
		FootnoteAnchorPrefix:       prefix,
		FootnoteReturnLinkContents: `&#8617;`,
	}

	renderer := blackfriday.HtmlRendererWithParameters(htmlFlags, "", "", parameters)

	// set up the parser
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_HEADER_IDS
	extensions |= blackfriday.EXTENSION_FOOTNOTES

	content := blackfriday.Markdown(p.Content, renderer, extensions)

	return template.HTML(content)
}

func LoadPostsFromPath(postPath string, readContent bool) (PostList, error) {
	var outerErr error = nil
	postList := make(PostList, 0, 15)
	if glog.V(1) {
		glog.Infoln("LoadPostsFromPath: Loading from", postPath)
	}
	err := filepath.Walk(postPath,
		func(filePath string, info os.FileInfo, err error) error {
			if info == nil {
				return os.ErrNotExist
			}
			if info.IsDir() ||
				path.Base(filePath)[0] == '.' ||
				!strings.HasSuffix(filePath, ".md") {
				// Skip directories, files starting with dot, and non-MD files.
				return nil
			}
			if glog.V(1) {
				glog.Infoln("LoadPostsFromPath: Loading", filePath)
			}
			newPost, err := NewPost(filePath, readContent)
			if err == nil {
				postList = append(postList, newPost)
			} else {
				glog.Errorf("Failed parsing post at %s: %s", filePath, err)
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
		glog.Warningln("Nothing in posts directory")
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
			glog.Errorln("NewArchiveSpecList: Failed to open", yearDirPath)
			continue
		}

		monthDirs, err := yearDir.Readdir(0)
		if err != nil {
			glog.Errorln("NewArchiveSpecList: Failed to read files from", yearDirPath)
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

	var sortObj sort.Interface = list
	if config.ArchiveListNewestFirst {
		sortObj = sort.Reverse(sortObj)
	}
	sort.Sort(sortObj)

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
