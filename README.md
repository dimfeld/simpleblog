simpleblog [![Build Status](https://travis-ci.org/dimfeld/simpleblog.png?branch=master)](https://travis-ci.org/dimfeld/simpleblog)
===========

A very simple blog engine I'm writing to learn Go and play with other techniques in a simple environment. This is not yet finished.

This blog engine generates and serves static pages, with tag and archiving support. Yes, such things already exist and will probably do a better job, but where's the fun in that?

While writing this project, I created a number of useful libraries for [caching](https://github.com/dimfeld/gocache), quick [HTTP routing]((https://github.com/dimfeld/httptreemux), simultaneous [configuration loading](https://github.com/dimfeld/goconfig) from TOML files and environment variables, and [more]((https://github.com/dimfeld).

### Acknowledgements

* [blackfriday](https://github.com/russross/blackfriday) for Markdown->HTML conversion.
* [fsnotify](https://github.com/howeyc/fsnotify) for inotify support.