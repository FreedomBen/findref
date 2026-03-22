# fr / findref



Here you can download older releases of the findref tool.

Current Release Version: 1.6.1

| Version | Linux | macOS | Windows | FreeBSD | OpenBSD |
|:-------:|:-----:|:-----:|:-------:|:-------:|:--------|
| 1.6.1 | [386](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/linux/386/findref.zip) - [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/linux/amd64/findref.zip) - [arm](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/linux/arm/findref.zip) - [arm64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/linux/arm64/findref.zip) | [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/darwin/amd64/findref.zip) - [arm64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/darwin/arm64/findref.zip) | [386](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/windows/386/findref.zip) - [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/windows/amd64/findref.zip) | [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/freebsd/amd64/findref.zip) - [arm64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/freebsd/arm64/findref.zip) | [amd64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/openbsd/amd64/findref.zip) - [arm64](https://raw.githubusercontent.com/FreedomBen/findref-bin/master/1.6.1/openbsd/arm64/findref.zip) |

## Linux packages

Each `linux/<arch>` directory listed above also contains ready-to-install packages for the major distributions. Build a download URL with `https://raw.githubusercontent.com/FreedomBen/findref-bin/master/<release>/linux/<arch>/<filename>` where `<release>` is one of the versions in the table and `<arch>` is `amd64`, `386`, `arm`, or `arm64` (the Arch package lives under `amd64`).

| Format | File name pattern | Package architectures |
|:-------|:------------------|:----------------------|
| Debian / Ubuntu (`.deb`) | `findref_<release>_<deb-arch>.deb` | `amd64`, `arm64`, `armhf`, `i386` |
| RPM distros (`.rpm`) | `findref-<release>-1.<rpm-arch>.rpm` | `x86_64`, `aarch64`, `armv7hl`, `i386` |
| Alpine (`.apk`) | `findref-<release>.<apk-arch>.apk` | `x86_64`, `aarch64`, `armhf`, `x86` |
| Arch Linux (`.pkg.tar.zst`) | `findref-<release>-1-x86_64.pkg.tar.zst` | `x86_64` |

Use the `latest` alias in place of `<release>` if you simply want the most recent version without updating scripts, or pick a specific release to pin an environment.
