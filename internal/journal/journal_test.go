package journal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"zakirullin/stuffbot/internal/userconfig"
)

func Test_insertDailyNote(t *testing.T) {
	r := require.New(t)
	now = func() time.Time {
		return time.Date(2023, 05, 30, 10, 04, 36, 0, time.UTC)
	}

	type testcase struct {
		name                string
		md                  string
		note                string
		want                string
		journalHeaderFormat string
	}

	tests := []testcase{
		{
			name: "Empty MD",
			note: "note 1",
			want: "#### 30, Tuesday\n\nnote 1\n",
		},
		{
			name: "No Headers",
			md:   "some text",
			note: "note 1",
			want: "some text\n\n#### 30, Tuesday\n\nnote 1\n",
		},
		{
			name: "Bare header",
			md:   "#### 30, Tuesday\n",
			note: "note 1",
			want: "#### 30, Tuesday\n\nnote 1\n",
		},
		{
			name: "Bare headers",
			md:   "#### 30, Tuesday\n\n#### 31, Friday\n",
			note: "note 1",
			want: "#### 30, Tuesday\n\nnote 1\n\n#### 31, Friday\n",
		},
		{
			name: "New daily note",
			md:   "#### 29, Tuesday\n\nnote 1",
			note: "note 2",
			want: "#### 29, Tuesday\n\nnote 1\n\n#### 30, Tuesday\n\nnote 2\n",
		},
		{
			name: "Append daily note",
			md:   "#### 29, Tuesday\nnote 1\n\n#### 30, Tuesday\nnote 2",
			note: "note 3",
			want: "#### 29, Tuesday\n\nnote 1\n\n#### 30, Tuesday\n\nnote 2\n\n---\n\nnote 3\n",
		},
		{
			name: "Append daily in the middle of the document",
			md:   "#### 29, Tuesday\nnote 1\n\n#### 30, Tuesday\nnote 2\n\n#### 31, Tuesday\nnote 4\n",
			note: "note 3",
			want: "#### 29, Tuesday\n\nnote 1\n\n#### 30, Tuesday\n\nnote 2\n\n---\n\nnote 3\n\n#### 31, Tuesday\n\nnote 4\n",
		},
		{
			name: "Append daily note",
			md:   "#### 29, Tuesday\n\nnote 1\n\n#### 30, Tuesday\n\nnote 2\n",
			note: "note 3",
			want: "#### 29, Tuesday\n\nnote 1\n\n#### 30, Tuesday\n\nnote 2\n\n---\n\nnote 3\n",
		},
		{
			name:                "Append daily note with custom header format",
			md:                  "#### 29, Tuesday\n* note 1\n\n#### 30, Tuesday\nsome text\n* note 2",
			note:                "note 3",
			want:                "#### 29, Tuesday\n* note 1\n\n#### 30, Tuesday\n\nsome text\n* note 2\n\n#### 30.05.2023\n\nnote 3\n",
			journalHeaderFormat: "02.01.2006",
		},
		{
			name: "Higher Level Header",
			md:   "#### 30, Tuesday\n\nnote 1\n\n## Some Header\n\nnote 2\n",
			note: "note 3",
			want: "#### 30, Tuesday\n\nnote 1\n\n---\n\nnote 3\n\n## Some Header\n\nnote 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.journalHeaderFormat == "" {
				tt.journalHeaderFormat = userconfig.DefaultConfig.JournalHeaderFormat()
			}
			got := insertDailyNote(tt.md, tt.journalHeaderFormat, tt.note)
			r.Equal(tt.want, got)
		})
	}
}
