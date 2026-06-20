package embedui

import "embed"

// FS contains the built React dashboard assets embedded into the Rift binary.
//
//go:embed ui/*
var FS embed.FS
