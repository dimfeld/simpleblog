package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testContent string = `# This is some content
And some more content

## It was the best of times.

### It was the worst of times.
Charles Dickens Charles Dickens`

var testHTML []string = []string{
	"<h1>This is some content</h1>",
	"<p>And some more content</p>",
	"<h2>It was the best of times.</h2>",
	"<h3>It was the worst of times.</h3>",
	"<p>Charles Dickens Charles Dickens</p>",
}

var writeCapturer *WriteCapturer

func init() {
	writeCapturer = &WriteCapturer{}
	logger = log.New(writeCapturer, "testlog", log.LstdFlags)
	config = &Config{}
}

func testOnePost(t *testing.T, title, date, tags string, includePostHeaderLine bool,
	content string) {

	t.Logf("Testing %s,  %s, \"%s\", %s",
		title, date, tags, strconv.FormatBool(includePostHeaderLine))

	var expectedTime time.Time
	expectedTags := make(map[string]bool)
	var err error
	validPost := true

	f, err := ioutil.TempFile("", "testPost")
	if err != nil {
		t.Fatal("Creating test post:", err)
	}

	if title != "MISSING" {
		f.WriteString(title + "\n")
	} else {
		validPost = false
	}

	if date != "MISSING" {
		f.WriteString(date + "\n")
		expectedTime, err = time.Parse(PostTimeFormat, strings.TrimSpace(date))
		if err != nil {
			validPost = false
		}
	} else {
		validPost = false
	}

	if tags != "MISSING" {
		f.WriteString(tags + "\n")

		if tags != "" {
			tagsList := strings.Split(tags, ",")
			for _, tag := range tagsList {
				tag = strings.Title(strings.TrimSpace(tag))
				expectedTags[tag] = true
			}
		}
	} else {
		validPost = false
	}

	if includePostHeaderLine {
		f.WriteString("\n")
	} else {
		validPost = false
	}

	if content != "MISSING" {
		f.WriteString(content)
	} else {
		// Missing content is actually ok...
		content = ""
	}

	filename := f.Name()
	f.Close()

	defer os.Remove("filename")

	postWithContent, err := NewPost(filename, true)
	if validPost && err != nil {
		t.Error("Expected valid post but saw error", err)
		return
	} else if !validPost {
		if err == nil && postWithContent != nil {
			t.Error("Expected invalid post but parsing succeeded")
		}
		return
	}

	if postWithContent.SourcePath != filename {
		t.Errorf("Expected SourcePath %s, saw %s", filename, postWithContent.SourcePath)
	}

	expectedTitle := strings.TrimSpace(title)
	if postWithContent.Title != expectedTitle {
		t.Errorf("Expected title \"%s\", saw \"%s\"", expectedTitle, postWithContent.Title)
	}

	if !postWithContent.Timestamp.Equal(expectedTime) {
		t.Errorf("Expected time \"%s\", saw \"%s\"", expectedTime, postWithContent.Timestamp)
	}

	for _, tag := range postWithContent.Tags {
		_, ok := expectedTags[tag]
		if !ok {
			t.Error("Unexpected tag", tag)
		}
		delete(expectedTags, tag)
	}

	for tag := range expectedTags {
		t.Error("Did not see expected tag", tag)
	}

	if string(postWithContent.Content) != content {
		t.Errorf("Expected content: %s\nSaw content: %s", content)
	}

	htmlContent := string(postWithContent.HTMLContent(false))

	if len(content) != 0 {
		for _, expectedHTML := range testHTML {
			if !strings.Contains(htmlContent, expectedHTML) {
				t.Errorf("Expected HTML to contain: %s\nSaw HTML: %s", testHTML, htmlContent)
			}
		}
	}

	postWithoutContent, err := NewPost(filename, false)
	if err != nil {
		t.Error("Error loading post without content:", err)
	}

	if postWithoutContent.SourcePath != filename {
		t.Errorf("Post without content expected SourcePath %s, saw %s",
			filename, postWithoutContent.SourcePath)
	}

	if postWithoutContent.Title != expectedTitle {
		t.Errorf("Post without content expected title %s, saw %s",
			expectedTitle, postWithoutContent.Title)
	}

	if !postWithoutContent.Timestamp.Equal(expectedTime) {
		t.Errorf("Post without content expected time %s, saw %s",
			expectedTime, postWithoutContent.Timestamp)
	}

	for i := range postWithContent.Tags {
		if i >= len(postWithoutContent.Tags) {
			t.Errorf("Post without content is missing tag %d", i)
		} else if postWithContent.Tags[i] != postWithoutContent.Tags[i] {
			t.Errorf("Post without content expected tag #%d %s, has %s",
				i, postWithContent.Tags[i], postWithoutContent.Tags[i])
		}
	}

	if postWithoutContent.Content != nil {
		t.Error("Post without content had content", postWithoutContent.Content)
	}
}

func TestNewPost(t *testing.T) {
	titles := []string{"Valid Title", " Extra spaces ", "", "MISSING"}
	dates := []string{"10/12/14 4:15PM MST", "6/4/12 3:57AM CDT  ",
		"", "MISSING"}
	tags := []string{"onetag", "tag1,tag 2", "a long tag", "tag 1, tag2, tag 3", "", "MISSING"}
	contents := []string{testContent, "", "MISSING"}
	includePostHeaderLine := []bool{true, false}

	for _, title := range titles {
		for _, date := range dates {
			for _, tag := range tags {
				for _, include := range includePostHeaderLine {
					for _, content := range contents {
						testOnePost(t, title, date, tag, include, content)
						// if t.Failed() {
						// 	// Quit early if we fail to ease debugging.
						// 	t.FailNow()
						// }
					}
				}
			}
		}
	}

	f, err := ioutil.TempFile("", "temppost")
	if err != nil {
		t.Fatal("Failed to create temp file")
	}
	filename := f.Name()
	// Make the file unreadable.
	f.Chmod(0200)
	f.Close()
	defer os.Remove(filename)

	_, err = NewPost(filename, true)
	if err == nil {
		t.Error("No error returned when reading unreadable post")
	}

}

var testPosts []*Post

func createTestPosts(t *testing.T) {
	// Can't call time.Parse in a static initializer, so we make this here.
	testPosts = make([]*Post, 3)

	postTime, err := time.Parse(PostTimeFormat, "1/2/14 4:15PM MST")
	if err != nil {
		t.Fatal("Invalid time in createTestPosts #0")
	}
	testPosts[0] = &Post{"2014/02/test-post1.md",
		"TestPost1",
		postTime,
		[]string{"tag1", "tag2"},
		[]byte("content")}

	postTime, err = time.Parse(PostTimeFormat, "1/3/14 4:15PM MST")
	if err != nil {
		t.Fatal("Invalid time in createTestPosts #0")
	}
	testPosts[1] = &Post{"2014/02/test-post2.md",
		"TestPost2",
		postTime,
		[]string{"tag1"},
		[]byte("content")}

	postTime, err = time.Parse(PostTimeFormat, "1/2/12 4:15PM MST")
	if err != nil {
		t.Fatal("Invalid time in createTestPosts #0")
	}
	testPosts[2] = &Post{"2012/02/test-post3.md",
		"TestPost3",
		postTime,
		[]string{"tag2"},
		[]byte("content")}
}

func writePost(t *testing.T, postPath string, post *Post) {
	dir, _ := path.Split(post.SourcePath)
	dir = path.Join(postPath, dir)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Error making directory %s: %s", dir, err)
	}

	fullPath := path.Join(postPath, post.SourcePath)
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("Error creating %s: %s", fullPath, err)
	}
	defer f.Close()

	f.WriteString(post.Title + "\n")
	postTime := post.Timestamp.Format("1/2/06 3:04PM MST")
	f.WriteString(postTime + "\n")
	f.WriteString(strings.Join(post.Tags, ",") + "\n\n")
	f.Write(post.Content)
}

func createPostTree(t *testing.T) (dirPath string) {
	createTestPosts(t)
	dirPath, err := ioutil.TempDir("", "testposts")
	if err != nil {
		t.Fatal("Could not create temporary directory:", err)
	}

	for _, post := range testPosts {
		writePost(t, dirPath, post)
	}

	return dirPath
}

type WriteCapturer struct {
	data []string
}

func (w *WriteCapturer) Write(data []byte) (n int, err error) {
	if w.data == nil {
		w.Clear()
	}
	w.data = append(w.data, string(data))
	return len(data), nil
}

func (w *WriteCapturer) Clear() {
	w.data = make([]string, 0)
}

func TestLoadPostsFromPath(t *testing.T) {
	dir := createPostTree(t)
	defer os.RemoveAll(dir)

	checkPostList := func(postList PostList, expected []string) {
		if len(postList) != len(expected) {
			t.Errorf("Expected %d posts but found %d",
				len(expected), len(postList))
		}

		for i, exp := range expected {
			if postList[i].SourcePath != exp {
				t.Errorf("Expected post 0 to be %s, saw %s",
					exp, postList[i].SourcePath)
			}
		}

	}

	t.Log("Loading 3 valid posts")
	expectedSorted := []string{"2012/02/test-post3.md",
		"2014/02/test-post1.md",
		"2014/02/test-post2.md"}
	for i, val := range expectedSorted {
		expectedSorted[i] = path.Join(dir, val)
	}

	postList, err := LoadPostsFromPath(dir, true)
	sort.Sort(postList)

	checkPostList(postList, expectedSorted)

	t.Log("Test with a non-.md post")
	writeCapturer.Clear()
	nonMdPost := &Post{"2012/02/other-post.txt",
		"TestPost4",
		time.Now(),
		[]string{"tag2"},
		[]byte("content")}
	writePost(t, dir, nonMdPost)
	nonMdPost.SourcePath = "2012/02/.somepost.md"
	writePost(t, dir, nonMdPost)

	postList, err = LoadPostsFromPath(dir, true)
	sort.Sort(postList)
	checkPostList(postList, expectedSorted)

	t.Log("Testing with 3 valid posts and 1 invalid post")
	writeCapturer.Clear()
	ioutil.WriteFile(path.Join(dir, "2012/02/invalidpost.md"), []byte("Invalid post"), 0666)
	postList, err = LoadPostsFromPath(dir, true)
	if err == nil {
		t.Error("LoadPostsFromPath did not propagate error from NewPost")
	}
	if postList == nil || len(postList) != 3 {
		t.Error("LoadPostsFromPath did not load valid posts because of one invalid post.")
	}
	sort.Sort(postList)
	checkPostList(postList, expectedSorted)
	if len(writeCapturer.data) != 2 {
		t.Errorf("Expected 2 log messages, saw %d: %v", len(writeCapturer.data), writeCapturer.data)
	}

	t.Log("Testing load from invalid path")
	_, err = LoadPostsFromPath("/jklsdfjklfds hjlksdfj", true)
	if err == nil {
		t.Error("No error returned on loading from nonexistent path.")
	}
}

func TestArchiveSpecList(t *testing.T) {
	checkSpecList := func(expected, actual ArchiveSpecList) {
		if len(expected) != len(actual) {
			t.Errorf("Expected %d specs but saw %d",
				len(expected), len(actual))
			return
		}

		for i, spec := range expected {
			if !time.Time(spec).Equal(time.Time(actual[i])) {
				t.Errorf("Expected spec #%d %s, saw %s",
					i, spec, actual[i])
			}
		}
	}

	writeCapturer := &WriteCapturer{}
	logger = log.New(writeCapturer, "testlog", log.LstdFlags)

	dir := createPostTree(t)
	defer os.RemoveAll(dir)

	expectedSpecList := ArchiveSpecList{
		ArchiveSpec(time.Date(2012, time.Month(02), 1, 1, 1, 1, 1, time.UTC)),
		ArchiveSpec(time.Date(2014, time.Month(02), 1, 1, 1, 1, 1, time.UTC)),
	}

	t.Log("Test normal spec list creation")
	specList, err := NewArchiveSpecList(dir)
	if err != nil {
		t.Error("Error creating spec list")
	}
	sort.Sort(specList)
	checkSpecList(expectedSpecList, specList)

	href := specList[1].Href()
	if href != "/2014/02" {
		t.Errorf("Expected href %s, saw %s", "/2014/02", href)
	}

	text := specList[1].String()
	if text != "Feb 2014" {
		t.Errorf("Expected text %s, saw %s", "Feb 2014", "text")
	}

	t.Log("Test with some non-numeric directories")
	os.Mkdir(path.Join(dir, "abc"), 0755)
	os.Mkdir(path.Join(dir, "2012", "abc"), 0755)
	specList, err = NewArchiveSpecList(dir)
	sort.Sort(specList)
	checkSpecList(expectedSpecList, specList)

	t.Log("Test reverse sort")
	expectedSpecList[0], expectedSpecList[1] = expectedSpecList[1], expectedSpecList[0]
	sort.Sort(sort.Reverse(specList))
	checkSpecList(expectedSpecList, specList)

	_, err = NewArchiveSpecList("/jklsdfnkjlse kslef")
	if err == nil {
		t.Error("NewArchiveSpecList did not fail with invalid directory")
	}
}
