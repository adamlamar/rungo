# rungo

Simple shim to run the version of Go you need. Designed primarily for integration with automated build processes, but can fill development needs too.

## Example

```
$ rungo 1.8.1 version
time="2017-05-07T18:17:42-06:00" level=info msg="Downloading file https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz" 
time="2017-05-07T18:17:57-06:00" level=info msg="Successfully extracted "/home/alamar/.go/1.8.1/go1.8.1.linux-amd64.tar.gz"" 
go version go1.8.1 linux/amd64
```

On the first invocation, `rungo` downloads the binary distribution of Go 1.8.1, extracts the tarball to `~/.go/<version>/`, and executes `go version`. Once the desired version of Go is installed, `rungo` simply delegates to the appropriate `go` command.

```
$ rungo 1.8.1 version
go version go1.8.1 linux/amd64
```

Of course the `rungo` can invoke any `go` subcommand, including build, run, tool, etc.

That's it!

## Platforms

Supports Linux, Windows, and OSX. Other platforms may work, but are untested.

## Building

At a minimum, `rungo` can build on 1.5, but may work on prior versions. Note that `rungo` should be built using `go build` - using `go run main.go` will not work.
