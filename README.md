# rungo [![Build Status](https://travis-ci.org/adamlamar/rungo.svg?branch=master)](https://travis-ci.org/adamlamar/rungo)

Simple shim to run the version of Go you need. Different than version managers you've used in the past.

## Installation

```
brew install adamlamar/rungo/rungo
```

## Example

`rungo` replaces your `go`, `gofmt`, and `godoc` binaries. Immediately after installing you can pretend like it isn't even there:

```
$ go version
time="2018-05-15T18:36:39-06:00" level=info msg="Downloading file https://storage.googleapis.com/golang/go1.10.2.darwin-amd64.tar.gz"
time="2018-05-15T18:36:52-06:00" level=info msg="Successfully extracted \"/Users/alamar/.rungo/1.10.2/go1.10.2.darwin-amd64.tar.gz\""
go version go1.10.2 darwin/amd64
```

On the first invocation, `rungo` downloads the binary distribution of Go 1.10.2, extracts the tarball to `~/.rungo/<version>/`, and executes `go version`. All future invocations simply delegate to the appropriate `go` command:

```
$ go version
go version go1.10.2 darwin/amd64
```

`rungo` can invoke any `go` subcommand, including build, run, tool, etc.

## Version specification

To specify the desired `go` version, set either the `GO_VERSION` environment variable or use a `.go-version` file.

Environment variable:
```
$ GO_VERSION=1.9.2 go version
go version go1.9.2 darwin/amd64
```

To set a default version, use a `.go-version` file in your home directory:
```
$ echo "1.9.2" > $HOME/.go-version
$ go version
go version go1.9.2 darwin/amd64
```

Or commit to your git repository on a per-project basis:
```
$ cd path/to/my-project
$ echo "1.9.2" > .go-version
$ git commit .go-version
$ git push
# => All users of your project will now use the exact version specified
```

If no version is specified, `rungo` will invoke the latest stable golang release shipped as of this version of rungo.

## Platforms

Tested on Linux and OSX. Windows support is maintained on a best-effort basis. Other platforms may work, but are untested.

Currently, only OSX has package manager support using homebrew.

## Building

At a minimum, `rungo` can build on 1.5, but may work on prior versions. Note that `rungo` should be built using `go build` - using `go run main.go` will not work.

## Manual Installation on Linux

Once you have the `rungo` binary, copy it somewhere in your path like `/usr/local/bin`. Then symlink the 3 go commands to the `rungo` binary.
Like this:

```
# cp rungo /usr/local/bin
# ln -s rungo go
# ln -s rungo gofmt
# ln -s rungo godoc
```

Done! You should be able to invoke `go`, `gofmt`, and `godoc` as desired.

## FAQ

### What are the command line switches and arguments?
There are none.

### How can I turn on debug logging?
Set the environment variable `RUNGO_VERBOSE` to any value. Example: `RUNGO_VERBOSE=t go version`

### How do I recover disk space used by rungo?
Delete the versioned directory you don't need anymore in `~/.rungo/<version>`

### How do I clear rungo's cache completely?
`rm -rf ~/.rungo`

### How does rungo work?
`rungo` "replaces" the golang binaries that would normally reside in your `$PATH`. For each command that `rungo` instruments, a symlink is used to point back to the `rungo` binary. On startup, `rungo` reads the basename of the program (i.e. the symlink name) and uses that to determine which follow-on command should be invoked. After that, `rungo` determines the appropriate version (downloading if necessary) and exec's with the expected arguments.
