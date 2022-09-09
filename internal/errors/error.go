package errors

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"
)

func Log(function any, err error) {
	log.Printf("%s: %v", getFunctionName(function), err)
}

func Format(function any, err error) error {
	return fmt.Errorf("%s: %w", getFunctionName(function), err)
}

func getFunctionName(function any) string {
	return runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name()
}

func New(text string) error {
	return errors.New(text)
}
