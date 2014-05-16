package main

import (
	"bytes"
	"fmt"
	"github.com/dimfeld/gocache"
	"html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	Posts      []*Post
	Page       *Post
	Tags       TagPopularity
	Archives   ArchiveSpecList
	globalData *GlobalData
}

func (t TemplateData) HrefFromPostPath(p string) string {
	relPath, err := filepath.Rel(t.globalData.postsDir, p)
	if err != nil {
		relPath = path.Base(p)
	}
	return relPath[:len(relPath)-3]
}

func (ps PageSpec) Fill(cacheObj gocache.Cache, key string) (gocache.Object, error) {
	posts, err := ps.generator(ps.globalData, ps.params)
	if err != nil {
		return gocache.Object{}, err
	}

	if len(posts) == 0 {
		logger.Println("Empty post list for", key)
		// No error, but an empty post list means that no matching file was found.
		return gocache.Object{}, os.ErrNotExist
	}

	templateData := TemplateData{globalData: ps.globalData}
	if ps.customPage {
		templateData.Page = posts[0]
	} else {
		templateData.Posts = posts
	}

	if ps.globalData.archive == nil {
		archive, err := NewArchiveSpecList(ps.globalData.postsDir)
		if err != nil {
			panic(err)
		}

		ps.globalData.archive = archive
	}

	templateData.Archives = ps.globalData.archive
	tags := NewTags(ps.globalData.tagsPath, ps.globalData.postsDir)
	templateData.Tags = tags.TagsByPopularity()

	debugf("Fill: Got ArchiveList of length %d", len(ps.globalData.archive))

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
	postPath := path.Join(globalData.postsDir, params["year"], params["month"], params["post"]) + ".md"
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
	tags := NewTags(globalData.tagsPath, globalData.postsDir)
	tagName, err := url.QueryUnescape(params["tag"])
	postNames, ok := tags.Tag[tagName]
	if err != nil || !ok || len(postNames) == 0 {
		return nil, os.ErrNotExist
	}

	postList := make(PostList, len(postNames))
	for i := range postList {
		postList[i] = tags.Post[postNames[i]]
	}

	// Sort the post list in the configured order.
	var sortObj sort.Interface = postList
	if globalData.tagsPageReverseSort {
		sortObj = sort.Reverse(sortObj)
	}
	sort.Sort(sortObj)

	return postList, nil
}

func generateIndexPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	postList := make(PostList, 0, globalData.indexPosts)

	for _, current := range globalData.archive {
		postPath := PostPath(globalData.postsDir, current.Year(), current.Month())
		monthPosts, err := LoadPostsFromPath(postPath, true)
		if err != nil {
			return nil, err
		}

		debugf("generateIndexPage: Loaded %d posts from %s", len(monthPosts), postPath)
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

	return postList, nil
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
