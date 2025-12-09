<!-- go-what - README.md -->
<!-- Copyright (c) 2016 MIT PDOS -->
<!-- Copyright (c) 2025 Jeffrey H. Johnson -->
<!-- SPDX-License-Identifier: MIT-0 -->
<!-- scspell-id: 8ff6fa5e-83cd-11f0-9914-80ee73e9b8e7 -->
# go-what

A Go conversion of [mit-pdos/what](https://github.com/mit-pdos/what)

## Build

```sh
env GOTOOLCHAIN="$(grep '^go .*$' go.mod | tr -cd 'go0-9.\n')+auto" go build -v
```
