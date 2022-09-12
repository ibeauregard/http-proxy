package cache

import (
	"bufio"
	"io"
)

type cacheEntryReader struct {
	*bufio.Reader
	io.Closer
}
