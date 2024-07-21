package internal

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	"zakirullin/stuffbot/internal/fs"
)

type Configuration struct {
	StoragePath    string `required:"true" envconfig:"STORAGE_PATH"`
	BotAPIToken    string `required:"true" envconfig:"BOT_API_TOKEN"`
	ConfigFilename string `default:"config.json"`
	Host           string `default:"https://127.0.0.1"`
}

var Config Configuration

func LoadConfig() error {
	if err := envconfig.Process("", &Config); err != nil {
		return fmt.Errorf("can't load config: %w", err)
	}

	return nil
}

func shouldSplitChecklist(checklist string) bool {
	for _, unsplittableChecklist := range []string{fs.DirRead, fs.DirWatch} {
		if checklist == unsplittableChecklist {
			return false
		}
	}
	return true
}
