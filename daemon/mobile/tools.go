//go:build tools

// This file is never compiled (build tag "tools"). Its only purpose is to anchor
// golang.org/x/mobile in daemon/go.mod so `gomobile bind ./mobile` is reproducible and does
// not keep re-adding the dependency to a different module's go.mod. See mobile/BUILD.md.
package mobile

import _ "golang.org/x/mobile/bind"
