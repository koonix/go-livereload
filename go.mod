module github.com/koonix/go-livereload

// Require 1.22+ to benefit from the for-loop changes.
// https://tip.golang.org/doc/go1.22#language
go 1.22
// Use 1.24+ so we can use the runtime.AddCleanup function.
// https://tip.golang.org/doc/go1.24#improved-finalizers
toolchain go1.24.1

require golang.org/x/net v0.37.0
