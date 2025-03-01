// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package htmlpatch_test

import (
	"testing"

	"github.com/koonix/go-livereload/internal/htmlpatch"
)

func TestInsertScript(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		inputHTML  string
		outputHTML string
	}{
		{
			"blank",
			`myscript`,
			``,
			`<!DOCTYPE html><html><head><script>myscript</script></head><body></body></html>`,
		},
		{
			"orphan-text",
			`myscript`,
			`mytext`,
			`<!DOCTYPE html><html><head><script>myscript</script></head><body>mytext</body></html>`,
		},
		{
			"orphan-tag",
			`myscript`,
			`<p>myparagraph</p>`,
			`<!DOCTYPE html><html><head><script>myscript</script></head><body><p>myparagraph</p></body></html>`,
		},
		{
			"orphan-body",
			`myscript`,
			`<body key="value">mytext</body>`,
			`<!DOCTYPE html><html><head><script>myscript</script></head><body key="value">mytext</body></html>`,
		},
		{
			"no-head",
			`myscript`,
			`<html key="value"><body>lmao</body></html>`,
			`<!DOCTYPE html><html key="value"><head><script>myscript</script></head><body>lmao</body></html>`,
		},
		{
			"no-doctype",
			`myscript`,
			`<html key="value"><head key2="value2"><meta key3="value3"/></head><body>lmao</body></html>`,
			`<!DOCTYPE html><html key="value"><head key2="value2"><meta key3="value3"/><script>myscript</script></head><body>lmao</body></html>`,
		},
		{
			"full",
			`myscript`,
			`<!DOCTYPE mydoctype><html key="value"><head key2="value2"><meta key3="value3"/></head><body>lmao</body></html>`,
			`<!DOCTYPE mydoctype><html key="value"><head key2="value2"><meta key3="value3"/><script>myscript</script></head><body>lmao</body></html>`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputHTML, err := htmlpatch.InsertScript(
				[]byte(test.inputHTML),
				nil,
				test.script,
			)
			if err != nil {
				t.Fatalf("could not insert script into HTML: %s", err)
			}
			want := test.outputHTML
			got := string(outputHTML)
			if want != got {
				t.Errorf("incorrect output html; want %q, got %q", want, got)
			}
		})
	}
}
