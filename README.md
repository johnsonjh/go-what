<!-- go-what - README.md -->
<!-- Copyright (c) 2016 MIT PDOS -->
<!-- Copyright (c) 2025 Jeffrey H. Johnson -->
<!-- SPDX-License-Identifier: MIT-0 -->
<!-- scspell-id: 8ff6fa5e-83cd-11f0-9914-80ee73e9b8e7 -->
# go-what

A Golang conversion of [mit-pdos/what](https://github.com/mit-pdos/what)

## What?

`go-what` is an improved version of the `w` tool.  It finds all processes
associated with a TTY (not just those registered in `wtmp`), and reports
all users that are running anything.  In particular, unlike `w`, `go-what`
will also show things running in detached screens/tmuxen.

## Build

Working on Linux (and systems that provide a Linux-compatible procfs,
same as the original Python implementation):

```sh
env GOTOOLCHAIN="$(grep '^go .*$' go.mod | tr -cd 'go0-9.\n')+auto" \
  GOFLAGS="-ldflags=-s -w" \
    go build -v
```
