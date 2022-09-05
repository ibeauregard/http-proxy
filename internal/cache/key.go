package cache

import (
	"crypto/md5"
	"encoding/hex"
)

func GetKey(url string) string {
	checksum := md5.Sum([]byte(url))
	return hex.EncodeToString(checksum[:])
}
