// Package web embeds the static assets for the settings UI.
package web

import "embed"

//go:embed static/*
var StaticFiles embed.FS
