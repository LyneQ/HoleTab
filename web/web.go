// Package web exposes the embedded static file system.
package web

import "embed"

// StaticFiles holds the contents of the web/static directory, embedded at
// compile time. Use fs.Sub(StaticFiles, "static") to strip the prefix before
// serving.
//
//go:embed static/*
var StaticFiles embed.FS
