package cache

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestSysCreateSuccess(t *testing.T) {
	osFile := &os.File{}
	osCreate = func(name string) (*os.File, error) {
		return osFile, nil
	}
	file, err := sysCreate("foobar")
	assert.Nil(t, err)
	assert.Equal(t, osFile, file)
}

func TestSysCreateError(t *testing.T) {
	osCreate = func(name string) (*os.File, error) {
		return nil, errors.New("error")
	}
	file, err := sysCreate("foobar")
	assert.NotNil(t, err)
	assert.Nil(t, file)
}

func TestSysOpenSuccess(t *testing.T) {
	osFile := &os.File{}
	osOpen = func(name string) (*os.File, error) {
		return osFile, nil
	}
	file, err := sysOpen("foobar")
	assert.Nil(t, err)
	assert.Equal(t, osFile, file)
}

func TestSysOpenError(t *testing.T) {
	osOpen = func(name string) (*os.File, error) {
		return nil, errors.New("error")
	}
	file, err := sysOpen("foobar")
	assert.NotNil(t, err)
	assert.Nil(t, file)
}

func TestTimeDotNow(t *testing.T) {
	timeFunc = func() time.Time {
		return nowMock
	}
	assert.EqualValues(t, nowMock, timeDotNow())
}
