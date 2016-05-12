pprofdump
=========

This utility provides the ability for users unfamiliar with Go's toolchain
to retrieve a set of pprof profiles via HTTP and package them as a gzipped
tar file.


### Getting started

Download the latest prebuilt binaries of `pprofdump` for your operating system:

* [Mac OS X](https://github.com/benbjohnson/pprofdump/releases/download/v0.1.0/pprofdump-darwin-amd64.gz)
* [Linux](https://github.com/benbjohnson/pprofdump/releases/download/v0.1.0/pprofdump-linux-amd64.gz)
* [Windows](https://github.com/benbjohnson/pprofdump/releases/download/v0.1.0/pprofdump-windows-amd64.exe.gz)


### Usage

Once you have downloaded `pprofdump`, simply point it at the HTTP server
that has pprof endpoints available:

```sh
$ pprofdump http://localhost:1234 > mydump.tar.gz
```

Note that this will take 30 seconds because the CPU profile takes 30 seconds
to run. You can see verbose output using the `-v` flag.

You can also specify specific profiles you want to fetch:

```sh
$ pprofdump -profiles heap,goroutine http://localhost:1234
```

By default the tarball is written to stdout but you can redirect it using the
`-o` flag:

```sh
$ pprofdump -o /tmp/mydump.tar.gz http://localhost:1234
```
