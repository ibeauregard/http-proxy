package cache

import (
	"encoding/gob"
	"io"
	"os"
	"time"
)

var ioCopy = io.Copy
var afterFunc = time.AfterFunc
var sysRemove = os.Remove
var sysOpenFile = os.OpenFile
var osOpen = os.Open
var sysOpen = func(name string) (io.ReadWriteCloser, error) {
	return osOpen(name)
}
var osCreate = os.Create
var sysCreate = func(name string) (io.WriteCloser, error) {
	return osCreate(name)
}
var newEncoder = func(writer io.Writer) interface{ Encode(any) error } {
	return gob.NewEncoder(writer)
}
var newDecoder = func(reader io.Reader) interface{ Decode(any) error } {
	return gob.NewDecoder(reader)
}

var timeDotNow = time.Now
