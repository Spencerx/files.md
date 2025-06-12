package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall/js"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"

	"github.com/spf13/afero"

	"zakirullin/stuffbot/config"
	"zakirullin/stuffbot/i18n"
	"zakirullin/stuffbot/internal"
	"zakirullin/stuffbot/internal/db"
	"zakirullin/stuffbot/internal/fs"
	"zakirullin/stuffbot/internal/userconfig"
	"zakirullin/stuffbot/pkg/tg"
)

var (
	reply func(u internal.Update) error
	chat  *tg.FakeTG
)

type Update struct {
	Message string
	Command *tg.Cmd
}

type Response struct {
	Messages []tg.Message
}

func callAsync(funcName string, callback func(js.Value, error)) {
	promise := js.Global().Call(funcName)

	var successFunc, errorFunc js.Func

	successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer successFunc.Release()
		defer errorFunc.Release()
		callback(args[0], nil)
		return nil
	})

	errorFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer successFunc.Release() // Clean up both
		defer errorFunc.Release()
		callback(js.Undefined(), fmt.Errorf("error: %v", args[0]))
		return nil
	})

	promise.Call("then", successFunc).Call("catch", errorFunc)
}

func Reply(_ js.Value, args []js.Value) interface{} {
	callAsync("hi", func(result js.Value, err error) {
		if err != nil {
			sendResponse("Error: %v\n", err)
			return
		}
		sendResponse(result.String())
	})

	return nil
}

func sendResponse(vals ...any) {
	js.Global().Call("receiveResponse", vals...)
}

func main() {
	//initBot()

	js.Global().Set("reply", js.FuncOf(Reply))

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

	reply = func(u internal.Update) error {
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

func send(update Update) Response {
	if update.Command != nil {
		_ = reply(tg.NewUpdCmd(1, *update.Command))
	} else {
		_ = reply(tg.NewUpd(1, update.Message))
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

func newUpdate(message string, cmd *tg.Cmd) Update {
	return Update{
		Message: message,
		Command: cmd,
	}
}

func newCmd(name string, params []string) tg.Cmd {
	return tg.NewCmd(name, params)
}
