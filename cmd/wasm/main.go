package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall/js"

	"zakirullin/stuffbot/config"
	"zakirullin/stuffbot/i18n"
	"zakirullin/stuffbot/internal"
	"zakirullin/stuffbot/internal/db"
	"zakirullin/stuffbot/internal/fs"
	"zakirullin/stuffbot/internal/userconfig"
	"zakirullin/stuffbot/pkg/tg"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/spf13/afero"
)

var (
	updater func(u internal.Update) error
	chat    *tg.FakeTG
)

type Update struct {
	Message string
	Command *tg.Cmd
}

type Response struct {
	Messages []tg.Message
}

func main() {
	initBot()
	NewApp()

	js.Global().Set("hi", js.FuncOf())

	select {}

}

func initBot() {
	opts := &tint.Options{
		Level: slog.LevelDebug,
	}
	logger := slog.New(tint.NewHandler(os.Stderr, opts))
	slog.SetDefault(logger)

	// For GUI app we don't have required .env params
	_ = godotenv.Load()
	err := config.LoadGUIConfig()
	if err != nil {
		panic(fmt.Sprintf("Error loading cfg: %s\n", err))
	}

	// TODO move to embed
	err = i18n.LoadLangFile("i18n/ru.json")
	if err != nil {
		panic(fmt.Sprintf("Error loading i18n: %s\n", err))
	}

	updater = func(u internal.Update) error {
		defer func() {
			err := recover()
			if err != nil {
				debug.PrintStack()
				slog.Error("Bot panic", "err", err)
			}
		}()

		userID := u.UserID()

		userPath := config.GUICfg.GUIUserStoragePath
		userPath, err = filepath.Abs(userPath)
		if err != nil {
			slog.Error("Bot error: can't get absolute path for curent dir", "err", err)
			return err
		}
		userFS, err := fs.NewFS(userPath, afero.NewOsFs())
		if err != nil {
			slog.Error("Bot error: can't create fs", "err", err)
			return err
		}
		err = userFS.CreateDirsIfNotExist()
		if err != nil {
			slog.Error("Bot error: can't create user dirs", "err", err)
			return err
		}

		confFilename := config.GUICfg.ConfigFilename
		userconf := userconfig.NewConfig(userFS, userID, confFilename)
		err = userconf.CreateDefaultIfNotExists()
		if err != nil {
			slog.Error("Bot error: can't create default user config", "err", err)
			return err
		}

		if chat == nil {
			chat = tg.NewFakeTG()
		}
		bot := internal.NewBot(userID, chat, userFS, db.NewDB(userID), userconf)
		if err := bot.Reply(u); err != nil {
			slog.Error("Bot error", "err", err)
		}

		return nil
	}
}

func Send(update Update) Response {
	if update.Command != nil {
		_ = updater(tg.NewUpdCmd(1, *update.Command))
	} else {
		_ = updater(tg.NewUpd(1, update.Message))
	}

	var r Response
	r.Messages = chat.Messages
	if chat.EditedMessages != nil {
		r.Messages = append(r.Messages, chat.EditedMessages...)
	}

	chat.Messages = nil
	chat.EditedMessages = nil

	return r
}

func NewUpdate(message string, cmd *tg.Cmd) Update {
	return Update{
		Message: message,
		Command: cmd,
	}
}

func NewCmd(name string, params []string) tg.Cmd {
	return tg.NewCmd(name, params)
}
