simpleblog [![Build Status](https://travis-ci.org/dimfeld/simpleblog.png?branch=master)](https://travis-ci.org/dimfeld/simpleblog)
===========

A very simple blog engine I wrote to learn Go and play with other techniques in a simple environment.

This blog engine generates and serves static pages and an Atom feed, with tag and archive support. Yes, such things already exist and will probably do a better job, but where's the fun in that?

While writing this project, I created a number of useful libraries for [caching](https://github.com/dimfeld/gocache), quick [HTTP routing](https://github.com/dimfeld/httptreemux), simultaneous [configuration loading](https://github.com/dimfeld/goconfig) from TOML files and environment variables, and [more](https://github.com/dimfeld).

### Example

```
% go get github.com/dimfeld/simpleblog
% cd $GOPATH/github.com/dimfeld/simpleblog
% go build
% ./simpleblog simpleblog.conf.sample

Direct your browser to http://localhost:8080
```

The default template and CSS are intentionally minimal, but should function as an easy skeleton to add your own styling. You can also check out [my blog](http://www.danielimfeld.com) to see it in action.

### Acknowledgements

* [blackfriday](https://github.com/russross/blackfriday) for Markdown->HTML conversion.
* [fsnotify](https://github.com/howeyc/fsnotify) for inotify support.
* My [modified fork](https://github.com/dimfeld/glog) of [Zenoss's glog fork](https://github.com/zenoss/glog) for better logging.
