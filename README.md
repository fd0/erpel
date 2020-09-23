[![Build Status](https://travis-ci.org/fd0/erpel.svg?branch=master)](https://travis-ci.org/fd0/erpel)

# erpel

Filter log messages and only print those which did not match any filters.

# Installation

erpel requires Go version 1.11 or newer. To build `erpel`, run the following command:

```shell
$ go build
```

Afterwards please find a binary `erpel` in the current directory.

# Compatibility

erpel follows [Semantic Versioning](http://semver.org) to clearly define which
versions are compatible. The configuration file and command-line parameters and
user-interface are considered the "Public API" in the sense of Semantic
Versioning.
