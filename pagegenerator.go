package main

import (
	"bytes"
	"fmt"
	"github.com/dimfeld/glog"
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

type PageGenerator func(*GlobalData, map[string]string) (posts PostList, title string, err error)

type PageSpec struct {
	globalData     *GlobalData
	customPage     bool
	customTemplate string
	generator      PageGenerator
	params         map[string]string
}

type ArchiveSpec time.Time
type ArchiveSpecList []ArchiveSpec

type TemplateData struct {
	// Set Posts to make a list of posts. A custom page should set Page.
	Posts       []*Post
	Page        *Post
	WindowTitle string
	Tags        TagPopularity
	Archives    ArchiveSpecList
	Domain      string
	globalData  *GlobalData
}

func HrefFromPostPath(p string) template.HTML {
	relPath, err := filepath.Rel(config.PostsDir, p)
	if err != nil {
		relPath = path.Base(p)
	}
	return template.HTML("/" + relPath[:len(relPath)-3])
}

func FormatTime(timestamp time.Time) template.HTML {
	return template.HTML(timestamp.Format("January 2, 2006 3:04PM"))
}

func AtomTime(timestamp time.Time) template.HTML {
	return template.HTML(timestamp.Format(time.RFC3339))
}

func AtomNow() template.HTML {
	return template.HTML(time.Now().Format(time.RFC3339))
}

func AtomFeedRef() template.HTML {
	return template.HTML(fmt.Sprintf("http://%s/", config.Domain))
}

func AtomPostRef(post *Post) template.HTML {
	return template.HTML(fmt.Sprintf("http://%s%s",
		config.Domain,
		HrefFromPostPath(post.SourcePath),
	))
}

func XMLEncoding() template.HTML {
	return `<?xml version="1.0" encoding="utf-8"?>`
}

var templateFuncs = template.FuncMap{
	"HrefFromPostPath": HrefFromPostPath,
	"FormatTime":       FormatTime,
	"AtomTime":         AtomTime,
	"AtomNow":          AtomNow,
	"AtomFeedRef":      AtomFeedRef,
	"AtomPostRef":      AtomPostRef,
	"XMLEncoding":      XMLEncoding,
	// Open and closing double brace, for when these are needed in the template.
	"odb": func() template.HTML { return template.HTML("{{") },
	"cdb": func() template.HTML { return template.HTML("}}") },
}

func createTemplates() (*template.Template, error) {
	tem := template.New("main").Funcs(templateFuncs)
	return tem.ParseGlob(path.Join(config.DataDir, "templates/*.tmpl.html"))
}

func (ps PageSpec) Fill(cacheObj gocache.Cache, key string) (gocache.Object, error) {
	ps.globalData.RLock()
	archive := ps.globalData.archive
	ps.globalData.RUnlock()
	if archive == nil {

		archive, err := NewArchiveSpecList(config.PostsDir)
		if err != nil {
			panic(err)
		}

		ps.globalData.Lock()
		ps.globalData.archive = archive
		ps.globalData.Unlock()
	}

	posts, title, err := ps.generator(ps.globalData, ps.params)
	if err != nil {
		return gocache.Object{}, err
	}

	if len(posts) == 0 {
		glog.Warningln("Empty post list for", key)
		// No error, but an empty post list means that no matching file was found.
		return gocache.Object{}, os.ErrNotExist
	}

	templateData := TemplateData{
		globalData:  ps.globalData,
		Domain:      config.Domain,
		WindowTitle: title,
	}
	if ps.customPage {
		templateData.Page = posts[0]
	} else {
		templateData.Posts = posts
	}

	templateData.Archives = ps.globalData.archive
	tags := NewTags(config.TagsPath, config.PostsDir)
	templateData.Tags = tags.TagsByPopularity()

	if glog.V(2) {
		glog.Infof("Fill: Got ArchiveList of length %d", len(ps.globalData.archive))
	}

	buf := &bytes.Buffer{}
	ps.globalData.RLock()
	templates := ps.globalData.templates
	ps.globalData.RUnlock()

	templateName := ps.customTemplate
	if templateName == "" {
		templateName = "main.tmpl.html"
	}
	templates.ExecuteTemplate(buf, templateName, templateData)

	uncompressed, compressed, err := gocache.CompressAndSet(cacheObj, key, buf.Bytes(), time.Now())
	if strings.HasSuffix(key, ".gz") {
		return compressed, err
	} else {
		return uncompressed, err
	}
}

func generatePostPage(globalData *GlobalData, params map[string]string) (PostList, string, error) {
	postPath := path.Join(config.PostsDir, params["year"], params["month"], params["post"]) + ".md"
	post, err := NewPost(postPath, true)
	if err != nil {
		return nil, "", err
	}
	postList := PostList{post}

	return postList, post.Title, nil
}

func generateArchivePage(globalData *GlobalData, params map[string]string) (PostList, string, error) {
	archivePath := path.Join(config.PostsDir, params["year"], params["month"])
	posts, err := LoadPostsFromPath(archivePath, true)
	if err != nil && len(posts) == 0 {
		return nil, "", err
	}
	sort.Sort(posts)

	title := posts[0].Timestamp.Format("Jan 2006")
	return posts, title, nil
}

func generateTagsPage(globalData *GlobalData, params map[string]string) (PostList, string, error) {
	tags := NewTags(config.TagsPath, config.PostsDir)
	tagName, err := url.QueryUnescape(params["tag"])
	postNames, ok := tags.Tag[tagName]
	if err != nil || !ok || len(postNames) == 0 {
		return nil, "", os.ErrNotExist
	}

	postList := make(PostList, len(postNames))
	for i := range postList {
		postList[i] = tags.Post[postNames[i]]
	}

	// Sort the post list in the configured order.
	var sortObj sort.Interface = postList
	if config.TagsPageNewestFirst {
		sortObj = sort.Reverse(sortObj)
	}
	sort.Sort(sortObj)

	return postList, strings.Title(tagName), nil
}

func generateIndexPage(globalData *GlobalData, params map[string]string) (PostList, string, error) {
	postList := make(PostList, 0, config.IndexPosts)

	for _, current := range globalData.archive {
		postPath := PostPath(config.PostsDir, current.Year(), current.Month())
		monthPosts, _ := LoadPostsFromPath(postPath, true)

		if glog.V(1) {
			glog.Infof("generateIndexPage: Loaded %d posts from %s", len(monthPosts), postPath)
		}
		if len(monthPosts) != 0 {
			postList = append(postList, monthPosts...)
		}

		// We have enough posts.
		if len(postList) >= config.IndexPosts {
			break
		}
	}

	// Sort posts, starting with the most recent.
	sort.Sort(sort.Reverse(postList))

	if len(postList) > config.IndexPosts {
		postList = postList[0:config.IndexPosts]
	}

	// Return a blank title, which means to use the default.
	return postList, "", nil
}

func generateCustomPage(globalData *GlobalData, params map[string]string) (PostList, string, error) {
	pagePath := path.Join(config.PostsDir, "page", params["page"]) + ".md"
	post, err := NewPost(pagePath, true)
	if err != nil {
		return nil, "", err
	}

	return PostList{post}, post.Title, nil
}

func (l ArchiveSpecList) Less(i, j int) bool {
	return time.Time(l[i]).Before(time.Time(l[j]))
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
