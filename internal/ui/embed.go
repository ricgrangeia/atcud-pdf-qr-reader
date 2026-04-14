// Package ui embeds the single-page web interface into the binary.
// Placing the embed here keeps the path constraint simple:
// //go:embed only allows paths relative to the file — no ".." allowed.
package ui

import _ "embed"

// IndexHTML is the compiled web page served at GET /.
//
//go:embed index.html
var IndexHTML []byte
