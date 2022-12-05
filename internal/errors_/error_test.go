package errors_

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"my_proxy/internal/tests"
	"strings"
	"testing"
)

var packagePath = "my_proxy/internal/errors_"

func TestLog(t *testing.T) {
	errorMessage := "my error message"
	expectedLog := fmt.Sprintf("%s.Log: %s\n", packagePath, errorMessage)
	assert.True(t, strings.Contains(
		tests.CaptureLog(func() { Log(Log, New(errorMessage)) }),
		expectedLog,
	))
}

func TestFormat(t *testing.T) {
	err := New("error")
	assert.True(t, errors.Is(Format(Format, err), err))
}

func TestGetFunctionName(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("%s.TestGetFunctionName", packagePath), getFunctionName(TestGetFunctionName))
	assert.Equal(t, fmt.Sprintf("%s.getFunctionName", packagePath), getFunctionName(getFunctionName))
}

func TestNew(t *testing.T) {
	errorText := "my error message"
	assert.Equal(t, errorText, New(errorText).Error())
}
