# vfs [![Travis-CI](https://travis-ci.com/worldiety/vfs.svg?branch=master)](https://travis-ci.com/worldiety/vfs) [![Go Report Card](https://goreportcard.com/badge/github.com/worldiety/vfs)](https://goreportcard.com/report/github.com/worldiety/vfs) [![GoDoc](https://godoc.org/github.com/worldiety/vfs?status.svg)](http://godoc.org/github.com/worldiety/vfs) [![Sourcegraph](https://sourcegraph.com/github.com/worldiety/vfs/-/badge.svg)](https://sourcegraph.com/github.com/worldiety/vfs?badge) [![Coverage](http://gocover.io/_badge/github.com/worldiety/vfs)](http://gocover.io/github.com/worldiety/vfs) 
Another virtual filesystem abstraction for go.

Actually I've used and written already quite a few abstraction layers in a variety of languages and
because each one get's better than the predecessor and Go lacks
a vfs which fulfills our needs, a new one has to see the light of the day.

Design goals of our vfs implementation are a clear and well designed API which is not only easy to use but also
to implement. At the other hand a simple API usually comes with a compromise in either expressiveness or usability and
our VFS makes no exemptions: as usual our API is based on the experiences we made and is therefore highly opinionated,
which is something your hear everywhere in the go ecosystem. Therefore we only optimized the use cases
we had in mind and not *every* possible scenario. So if you require an API change, please don't be disappointed if
we may reject it, even if it is a perfect solution in a very specific scenario.  


# Status
This library is still in alpha and it's API has not been stabilized yet. As soon as this happens, there will be no
incompatible structrual API changes anymore. However the CTS profiles will be updated and refined over time.

# Available implementations

## FilesystemDataProvider

`import github.com/worldiety/vfs`

| CTS Check     | Result        |
| ------------- | ------------- |
| Empty|:white_check_mark: |
| Write any|:white_check_mark: |
| Read any|:white_check_mark: |
| Write and Read|:white_check_mark: |
| Rename|:white_check_mark: |