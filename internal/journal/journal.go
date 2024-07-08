package journal

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"zakirullin/stuffbot/internal/fs"
	pkgText "zakirullin/stuffbot/pkg/text"
)

var now = time.Now // to be replaced in tests

var newLines = regexp.MustCompile(`\n+`)

const (
	headerLevel = 4
)

func AddRecord(dir, noteFilename string, botFs *fs.FS) error {
	record, err := botFs.RestoreContent(dir, noteFilename)
	if err != nil {
		return fmt.Errorf("failed to move to journal: can't get note content: %w", err)
	}

	//time.Now().Format("`13:01`")

	journalFilename := now().Format("2024 January.md")
	exists, err := botFs.Exists(fs.DirJournal, journalFilename)
	if err != nil {
		return err
	}

	var md string
	if exists {
		md, err = botFs.Content(fs.DirJournal, journalFilename)
		if err != nil {
			return err
		}
		md = pkgText.NormNewLines(md)
		md = strings.TrimSpace(md)
	}

	header := fmt.Sprintf("### %d %s", time.Now().Day(), time.Now().Month())
	if !strings.Contains(md, header) {
		md = fmt.Sprintf("%s\n%s")
	}

	md = fmt.Sprintf("%s\n%s", md, record)

	return botFs.Put(fs.DirJournal, journalFilename, md)
}
