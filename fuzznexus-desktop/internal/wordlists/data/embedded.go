package data

import "embed"

// BuiltinPacksFS contains all gzip-compressed curated wordlist packs embedded into the binary.
//
//go:embed packs/*.json.gz
var BuiltinPacksFS embed.FS
