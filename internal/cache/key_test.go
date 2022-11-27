package cache

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetKey(t *testing.T) {
	tests := []struct {
		arg            string
		expectedOutput string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"a", "0cc175b9c0f1b6a831c399e269772661"},
		{"A", "7fc56270e7a70fa81a5935b72eacbe29"},
		{"aA", "8c80b057bc0b599b48cbd144558aeada"},
		{"alpha", "2c1743a391305fbf367df8e4f069f9f9"},
		{"omega", "c6d6bd7ebf806f43c76acc3681703b81"},
		{"https://www.google.com/", "d75277cdffef995a46ae59bdaef1db86"},
		{"https://www.twitter.com/", "a3eb9690c501f769a3c4de97ddc931d5"},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("GetKey(%q)", test.arg)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expectedOutput, GetKey(test.arg))
		})
	}
}
