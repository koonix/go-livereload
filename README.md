# go-livereload

go-livereload is a Go library
that adds live reloading capability
to any http handler.

## Usage

Import:

```go
package main

import "github.com/koonix/go-livereload"
```

Serve a directory:

```go
upstream := http.FileServer(http.Dir("frontend"))
lr := livereload.New(upstream)
http.ListenAndServe(":8090", lr)
```

Proxy another webserver:

```go
u, _ := url.Parse("http://localhost:8080")
upstream := livereload.ReverseProxy(u)
lr := livereload.New(upstream)
http.ListenAndServe(":8090", lr)
```

Reload the webpages open in browsers:

```go
lr.Reload()
```
