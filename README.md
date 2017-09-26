# fr / findref



findref is

## Why use this tool instead of X?



## Usage



## Installing

### Use the install script

If you are on an intel-based linux or mac, there is an install script located at
`install.sh`.  If on ARM or Windows, you should probably download the
[pre-built binary](#pre-built-binaries) below.

To let the script do the work, run this command.  Make sure to add `sudo` if
installing to a location that isn't writeable by your normal user:

# In your home directory (Make sure this destination is in your PATH variable)
```bash
curl -s https://raw.githubusercontent.com/FreedomBen/findref/master/install.sh | bash $HOME/bin
```

# Systemwide for all users (requires root access)
```bash
curl -s https://raw.githubusercontent.com/FreedomBen/findref/master/install.sh | sudo bash /usr/local/bin
```

### Pre-built binaries

If you wish, you can download prebuilt binaries for your system.  After downloading,
put it somewhere in your PATH.  I recommend /usr/local/bin:

Current Release Version: 0.0.2

| Release | Linux                     | macOS | Windows |
|:-------:|:-------------------------:|:-------------:|:-------------:|
| 0.0.2 | [arm](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/linux/arm/findref) - [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/linux/amd64/findref) - [arm64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/linux/arm64/findref) - [386](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/linux/386/findref) | [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/darwin/amd64/findref) - [386](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/darwin/386/findref) | [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/windows/amd64/findref.exe) - [386](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/0.0.2/windows/386/findref.exe) |

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

Pretty easy, tho that will build for every supported platform.  You can find the binary you
care about by looking in the `findref-bin` subdirectory and following the directory structure
until you find the correct binary for your system.

You can also build for just
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
