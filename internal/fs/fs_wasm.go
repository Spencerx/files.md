//go:build wasm

package fs

import (
	"os"
)

var Ctime = func(fi os.FileInfo) int64 {
	return 0
}
