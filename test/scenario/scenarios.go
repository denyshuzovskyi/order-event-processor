package scenario

import "embed"

//go:embed *
var Files embed.FS

const Path = "test/scenario"
