package scenarios

import "embed"

// Files contains the official built-in Platform Lab scenarios.
//
//go:embed course-01/*.yaml
var Files embed.FS
