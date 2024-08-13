package txt

import (
	"fmt"
	"regexp"
	"strings"
)

// MarkdownToHtml converts user's markdown to Telegram-supported subset of HTML
// Telegram supported tags:
// <b>bold</b>, <strong>bold</strong>
// <i>italic</i>, <em>italic</em>
// <u>underline</u>, <ins>underline</ins>
// <s>strikethrough</s>, <strike>strikethrough</strike>, <del>strikethrough</del>
// <span class="tg-spoiler">spoiler</span>, <tg-spoiler>spoiler</tg-spoiler>
// <b>bold <i>italic bold <s>italic bold strikethrough <span class="tg-spoiler">italic bold strikethrough spoiler</span></s> <u>underline italic bold</u></i> bold</b>
// <a href="http://www.example.com/">inline URL</a>
// <a href="tg://user?id=123456789">inline mention of a user</a>
// <tg-emoji emoji-id="5368324170671202286">👍</tg-emoji>
// <code>inline fixed-width code</code>
// <pre>pre-formatted fixed-width code block</pre>
// <pre><code class="language-python">pre-formatted fixed-width code block written in the Python programming language</code></pre>
// <blockquote>Block quotation started\nBlock quotation continued\nThe last line of the block quotation</blockquote>
// <blockquote expandable>Expandable block quotation started\nExpandable block quotation continued\nExpandable block quotation continued\nHidden by default part of the block quotation started\nExpandable block quotation continued\nThe last line of the block quotation</blockquote>
func MarkdownToHtml(markdown string) string {
	// Define the regex patterns for different markdown elements
	codeBlockPattern := regexp.MustCompile("```([\\s\\S]*?)```")
	inlindeCodePattern := regexp.MustCompile("`([.+?]*?)`")
	boldPattern := regexp.MustCompile(`(\*\*|__)(.+?)(\*\*|__)`)
	italicPattern := regexp.MustCompile(`(\*|_)(.+?)(\*|_)`)
	headerPattern := regexp.MustCompile(`(#{1,6})\s*(.+)`)

	// Find and replace code blocks with placeholders
	codeBlocks := codeBlockPattern.FindAllStringSubmatch(markdown, -1)
	for i, codeBlock := range codeBlocks {
		placeholder := fmt.Sprintf("{{{CODEBLOCK_%d}}}", i)
		markdown = strings.Replace(markdown, codeBlock[0], placeholder, 1)
	}
	inlineCodeBlocks := inlindeCodePattern.FindAllStringSubmatch(markdown, -1)
	for i, inlineCode := range inlineCodeBlocks {
		placeholder := fmt.Sprintf("{{{INLINECODE_%d}}}", i)
		markdown = strings.Replace(markdown, inlineCode[0], placeholder, 1)
	}

	// Replace markdown elements with HTML tags
	markdown = boldPattern.ReplaceAllString(markdown, `<b>$2</b>`)
	markdown = italicPattern.ReplaceAllString(markdown, `<i>$2</i>`)
	markdown = headerPattern.ReplaceAllString(markdown, `<b>$2</b>`)

	// Replace placeholders with the original code blocks wrapped in appropriate tags
	for i, codeBlock := range codeBlocks {
		placeholder := fmt.Sprintf("{{{CODEBLOCK_%d}}}", i)
		codeHTML := fmt.Sprintf("<pre>%s</pre>", codeBlock[1])
		markdown = strings.Replace(markdown, placeholder, codeHTML, 1)
	}
	for i, inlineCode := range inlineCodeBlocks {
		placeholder := fmt.Sprintf("{{{INLINECODE_%d}}}", i)
		codeHTML := fmt.Sprintf("<code>%s</code>", inlineCode[1])
		markdown = strings.Replace(markdown, placeholder, codeHTML, 1)
	}
	
	return markdown
}
