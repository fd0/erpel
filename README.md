[![Build Status](https://travis-ci.org/fd0/erpel.svg?branch=master)](https://travis-ci.org/fd0/erpel)

# erpel

Filter log messages and only print those which did not match any filters.

# Installation

erpel requires Go version 1.4 or newer. To build dachs, run the following command:

```shell
$ go run build.go
```

Afterwards please find a binary `erpel` in the current directory.

# Development

erpel is developed using the build tool [gb](https://getgb.io). It can be installed by running the following command:

```shell
$ go get github.com/constabulary/gb/...
```

The program can be compiled using `gb` as follows:

```shell
$ gb build
```

The tests are run when calling `gb` as follows:
```shell
$ gb test
```

# Compatibility

erpel follows [Semantic Versioning](http://semver.org) to clearly define which
versions are compatible. The configuration file and command-line parameters and
user-interface are considered the "Public API" in the sense of Semantic
Versioning.
