# rungo

Simple shim to run the version of Go you need. Designed primarily for integration with automated build processes, but can fill development needs too.

## Example

```
$ GO_VERSION=1.9.2 rungo version
time="2017-11-25T23:50:48-07:00" level=info msg="Downloading file https://storage.googleapis.com/golang/go1.9.2.darwin-amd64.tar.gz"
time="2017-11-25T23:51:00-07:00" level=info msg="Successfully extracted \"/Users/alamar/.go/1.9.2/go1.9.2.darwin-amd64.tar.gz\""
go version go1.9.2 darwin/amd64
```

On the first invocation, `rungo` downloads the binary distribution of Go 1.9.2, extracts the tarball to `~/.go/<version>/`, and executes `go version`. Once the desired version of Go is installed, `rungo` simply delegates to the appropriate `go` command.

Future invocations delegate to the `go` command with no log output:

```
$ GO_VERSION=1.9.2 rungo version
go version go1.9.2 darwin/amd64
```

`rungo` can invoke any `go` subcommand, including build, run, tool, etc.

## Installation

```
go get github.com/adamlamar/rungo
```

`$GOPATH/bin` should be in your `$PATH`.

## Version specification

To specify the desired `go` version, set either the `GO_VERSION` environment variable or use a `.go-version` file.

Environment variable:
```
$ GO_VERSION=1.9.2 rungo version
go version go1.9.2 darwin/amd64
```

`.go-version` file:
```
$ echo "1.9.2" > $HOME/.go-version
$ rungo version
go version go1.9.2 darwin/amd64
```

Of course the `.go-version` file only needs to be written once. `rungo` searches for the first `.go-version` that exists from the current working directory upwards, so `.go-version` can be used on a per-project basis.

## Platforms

Supports Linux, Windows, and OSX. Other platforms may work, but are untested.

## Building

At a minimum, `rungo` can build on 1.5, but may work on prior versions. Note that `rungo` should be built using `go build` - using `go run main.go` will not work.
