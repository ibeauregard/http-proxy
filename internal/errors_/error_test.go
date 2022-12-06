package errors_

import (
	"errors"
	"fmt"
	"github.com/ibeauregard/http-proxy/internal/tests"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var packagePath = "github.com/ibeauregard/http-proxy/internal/errors_"

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
