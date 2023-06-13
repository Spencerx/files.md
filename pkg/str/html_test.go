package str

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkdownToHtml_Header(t *testing.T) {
	r := require.New(t)

	md := `# Header`
	html := MarkdownToHtml(md)

	r.Equal("<h1>Header</h1>\n", html)
}
