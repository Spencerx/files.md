package server

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"zakirullin/stuffbot/config"
)

var lock sync.RWMutex

func LogRename(time int64, oldPath, newPath string) {
	lock.Lock()
	defer lock.Unlock()

	file, err := os.OpenFile(path.Join(config.BotCfg.WorkingDir, "fslog"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	oldPath = url.QueryEscape(oldPath)
	newPath = url.QueryEscape(newPath)
	record := fmt.Sprintf("%d %s %s\n", time, oldPath, newPath)

	file.WriteString(record)
	file.Sync()
}

// ReadLog reads the file system log and returns a map of old paths to new paths.
// AfterTimestamp is inclusive.
func ReadLog(userID, afterTimestamp int64) map[string]string {
	lock.RLock()
	defer lock.RUnlock()

	file, err := os.Open(path.Join(config.BotCfg.WorkingDir, "fslog"))
	if err != nil {
		return nil
	}
	defer file.Close()

	logEntries := make(map[string]string)
	scanner := bufio.NewScanner(file)
	userPathPrefix := path.Join(config.BotCfg.StorageDir, fmt.Sprintf("%d", userID))
	for scanner.Scan() {
		line := scanner.Text()
		var timestamp int64
		var oldPath, newPath string
		n, err := fmt.Sscanf(line, "%d %s %s", &timestamp, &oldPath, &newPath)
		if err != nil || n != 3 || timestamp < afterTimestamp {
			continue
		}

		if !strings.HasPrefix(oldPath, userPathPrefix) || !strings.HasPrefix(newPath, userPathPrefix) {
			continue
		}

		logEntries[oldPath] = newPath
	}

	return logEntries
}
