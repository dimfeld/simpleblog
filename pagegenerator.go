package main

import (
	"fmt"
	"github.com/dimfeld/simpleblog/cache"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	//"io/ioutil"
	//"html/template"
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
	Tags     []string
	Archives ArchiveSpecList
}

func (ps PageSpec) Fill(cacheObj cache.Cache, path string) (cache.Object, error) {
	data, err := ps.generator(ps.globalData, ps.params)
	if err != nil {
		return cache.Object{}, err
	}

	// TODO Actually do templating here.
	output := data[0].Content

	uncompressed, compressed, err := cache.CompressAndSet(cacheObj, path, output, time.Now())
	if strings.HasSuffix(path, ".gz") {
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

	// Do templating

	return postList, nil
}

func generateArchivePage(globalData *GlobalData, params map[string]string) (PostList, error) {
	archivePath := path.Join(globalData.postsDir, params["year"], params["month"])
	posts, err := LoadPostsFromPath(archivePath, true)
	sort.Sort(posts)
	if err != nil {
		// 404 error
	}
	return nil, nil
}

func generateTagsPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	return nil, nil
}

func generateIndexPage(globalData *GlobalData, params map[string]string) (PostList, error) {
	current := time.Now()

	postPath := PostPath(globalData.postsDir, current.Year(), current.Month())
	postList, err := LoadPostsFromPath(postPath, true)
	if err != nil {
		return nil, err
	}
	monthPosts := postList

	for len(postList) < globalData.indexPosts {
		current, err = PreviousMonthDir(globalData.postsDir, monthPosts[0].Timestamp)
		postPath = PostPath(globalData.postsDir, current.Year(), current.Month())
		if err != nil {
			if os.IsNotExist(err) {
				// We're out of posts.
				break
			} else {
				// Some other error
				return nil, err
			}
		}

		monthPosts, err = LoadPostsFromPath(postPath, true)
		if err != nil {
			return nil, err
		}

		postList = append(postList, monthPosts...)
	}

	sort.Sort(sort.Reverse(postList))

	if len(postList) > globalData.indexPosts {
		postList = postList[0:globalData.indexPosts]
	}

	return nil, nil
}

func (l ArchiveSpecList) Less(i, j int) bool {
	// Reverse sort so the most recent is first.
	return time.Time(l[i]).After(time.Time(l[j]))
}

func (l ArchiveSpecList) Len(i, j int) int {
	return len(l)
}

func (l ArchiveSpecList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (a ArchiveSpec) Href() string {
	return fmt.Sprintf("/%04d/%02d", time.Time(a).Year(), time.Time(a).Month())
}

func (a ArchiveSpec) Text() string {
	return time.Time(a).Format("Jan 2006")
}
