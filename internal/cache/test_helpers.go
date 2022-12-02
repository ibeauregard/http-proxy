package cache

import (
	"time"
)

var nowMock = time.Date(2022, 11, 30, 23, 21, 43, 0, time.UTC)
var (
	updateCacheBackup  = updateCache
	ioCopyBackup       = ioCopy
	cacheDirNameBackup = cacheDirName
)
