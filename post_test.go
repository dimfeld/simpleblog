package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testContent string = `# This is some content
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

func testOnePost(t *testing.T, title, date, tags string, includePostHeaderLine bool,
	content string) {

	t.Logf("Testing title: %s, date: %s, tags: \"%s\", postHeaderLine: %s",
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
		expectedTime, err = time.Parse("1/2/06 3:04PM MST", strings.TrimSpace(date))
		if err != nil {
			validPost = false
		}
	} else {
		validPost = false
	}

	if tags != "MISSING" {
		f.WriteString(tags + "\n")

		tagsList := strings.Split(tags, ",")
		for _, tag := range tagsList {
			tag = strings.Title(strings.TrimSpace(tag))
			expectedTags[tag] = true
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

	htmlContent := string(postWithContent.HTMLContent())

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
	tags := []string{"onetag", "tag1, tag 2", "a long tag", "tag 1, tag2, tag 3", "", "MISSING"}
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
}
