package main

import (
	"bytes"
	"fmt"
	"github.com/dimfeld/gocache"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	//"io/ioutil"
	"html/template"
)

type PageGenerator func(*GlobalData, map[string]string) (PostList, error)

type PageSpec struct {
	globalData *GlobalData
	customPage bool
	generator  PageGenerator
	params     map[string]string
}

type ArchiveSpec time.Time
type ArchiveSpecList []ArchiveSpec

type TemplateData struct {
	// Set Posts to make a list of posts. A custom page should set Page.
	Posts    []*Post
	Page     *Post
	Tags     TagPopularity
	Archives ArchiveSpecList
}

func (ps PageSpec) Fill(cacheObj gocache.Cache, key string) (gocache.Object, error) {
	posts, err := ps.generator(ps.globalData, ps.params)
	if err != nil {
		return gocache.Object{}, err
	}

	if len(posts) == 0 {
		// No error, but an empty post list means that no matching file was found.
		return gocache.Object{}, os.ErrNotExist
	}

	templateData := TemplateData{}
	if ps.customPage {
		templateData.Page = posts[0]
	} else {
		templateData.Posts = posts
	}
	templateData.Archives = ps.globalData.archive
	tags := NewTags(ps.globalData.tagsPath, ps.globalData.postsDir)
	templateData.Tags = tags.TagsByPopularity()

	tem, err := template.ParseFiles(path.Join(string(ps.globalData.dataDir), "templates/main.tmpl.html"))
	if err != nil {
		panic("Error parsing template: " + err.Error())
	}

	buf := &bytes.Buffer{}
	tem.ExecuteTemplate(buf, "main.tmpl.html", templateData)

	uncompressed, compressed, err := gocache.CompressAndSet(cacheObj, key, buf.Bytes(), time.Now())
	if strings.HasSuffix(key, ".gz") {
		return compressed, err
	} else {
		return uncompressed, err
	}
}

func generatePostPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	postPath := path.Join(globalData.postsDir, params["year"], params["month"], params["post"])
	post, err := NewPost(postPath, true)
	if err != nil {
		return nil, err
	}
	postList := PostList{post}

	return postList, nil
}

func generateArchivePage(globalData *GlobalData, params map[string]string) (PostList, error) {
	archivePath := path.Join(globalData.postsDir, params["year"], params["month"])
	posts, err := LoadPostsFromPath(archivePath, true)
	if err != nil {
		return nil, err
	}
	sort.Sort(posts)

	return posts, nil
}

func generateTagsPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	return nil, nil
}

func generateIndexPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	if globalData.archive == nil {
		archive, err := NewArchiveSpecList(globalData.postsDir)
		if err != nil {
			return nil, err
		}

		globalData.archive = archive
	}

	postList := make(PostList, 0, globalData.indexPosts)

	for _, current := range globalData.archive {
		postPath := PostPath(globalData.postsDir, current.Year(), current.Month())
		monthPosts, err := LoadPostsFromPath(postPath, true)
		if err != nil {
			return nil, err
		}

		postList = append(postList, monthPosts...)

		// We have enough posts.
		if len(postList) >= globalData.indexPosts {
			break
		}
	}

	// Sort posts, starting with the most recent.
	sort.Sort(sort.Reverse(postList))

	if len(postList) > globalData.indexPosts {
		postList = postList[0:globalData.indexPosts]
	}

	return nil, nil
}

func generateCustomPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	pagePath := path.Join(globalData.postsDir, "page", params["page"])
	post, err := NewPost(pagePath, true)
	if err != nil {
		return nil, err
	}

	return PostList{post}, nil
}

func (l ArchiveSpecList) Less(i, j int) bool {
	// Reverse sort so the most recent is first.
	return time.Time(l[i]).After(time.Time(l[j]))
}

func (l ArchiveSpecList) Len() int {
	return len(l)
}

func (l ArchiveSpecList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (a ArchiveSpec) Href() string {
	return fmt.Sprintf("/%04d/%02d", a.Year(), a.Month())
}

func (a ArchiveSpec) String() string {
	return time.Time(a).Format("Jan 2006")
}

func (a ArchiveSpec) Month() time.Month {
	return time.Time(a).Month()
}

func (a ArchiveSpec) Year() int {
	return time.Time(a).Year()
}
