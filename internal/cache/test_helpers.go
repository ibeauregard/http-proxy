package cache

import (
	"bytes"
	"log"
	"os"
	"time"
)

var nowMock = time.Date(2022, 11, 30, 23, 21, 43, 0, time.UTC)
var (
	updateCacheBackup  = updateCache
	ioCopyBackup       = ioCopy
	cacheDirNameBackup = cacheDirName
)

func captureLog(f func()) string {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)
	defer func() { log.SetOutput(os.Stderr) }()
	f()
	return buf.String()
}
