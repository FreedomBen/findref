# fr / findref



findref is

## Installing

If you wish, you can download prebuilt binaries for your system.  After downloading,
put it somewhere in your PATH.  I recommend /usr/local/bin:

Current Release Version: 0.0.2

| Release | Linux                     | macOS | Windows |
|:-------:|:-------------------------:|:-------------:|:-------------:|
| 0.0.2 | [arm](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/linux/arm/findref?raw=true) - [amd64](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/linux/amd64/findref?raw=true) - [arm64](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/linux/arm64/findref?raw=true) - [386](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/linux/386/findref?raw=true) | [amd64](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/darwin/amd64/findref?raw=true) - [386](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/darwin/386/findref?raw=true) | [amd64](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/windows/amd64/findref.exe?raw=true) - [386](https://github.com/FreedomBen/findref-bin/blob/master/0.0.2/windows/386/findref.exe?raw=true) |

### Older releases

The full catalog of releases is available to download.  See [ARCHIVES.md](ARCHIVES.md)

### Building from source

To build from source you can either use the docker build wrapper, or build it directly on your system.

If you have your [Go environment](https://golang.org/doc/install) set up
already, you can build it directly from source:

```bash
go get github.com/FreedomBen/findref
go install findref
```

To use the docker build, the easiest way is to use the rake task:

```bash
rake
```

Pretty easy, tho that will build for every supported platform.  You can also build for just
your platform.  Specify your OS for the `GOOS` value, and your arch for `GOARCH`.  See [here
for a list of valid targets](https://stackoverflow.com/a/30068222/2062384).

Example for Linux x64 (amd64):

```bash
    docker run \
      --rm \
      --volume "$(pwd):/usr/src/findref" \
      --workdir "/usr/src/findref" \
      --env GOOS=linux \
      --env GOARCH=amd64 \
      golang:#{GO_VERSION} go build
```

Example for Linux x32 (386):

```bash
    docker run \
      --rm \
      --volume "$(pwd):/usr/src/findref" \
      --workdir "/usr/src/findref" \
      --env GOOS=linux \
      --env GOARCH=386 \
      golang:#{GO_VERSION} go build
```

Example for macOS x64 (amd64)

```bash
    docker run \
      --rm \
      --volume "$(pwd):/usr/src/findref" \
      --workdir "/usr/src/findref" \
      --env GOOS=darwin \
      --env GOARCH=amd64 \
      golang:#{GO_VERSION} go build
```
