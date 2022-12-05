package tests

import (
	"bytes"
	"log"
	"os"
)

func CaptureLog(f func()) string {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)
	defer func() { log.SetOutput(os.Stderr) }()
	f()
	return buf.String()
}
