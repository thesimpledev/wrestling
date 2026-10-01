package main

import "embed"

//go:embed data/wrestlers/*.yaml
var DefaultCards embed.FS
