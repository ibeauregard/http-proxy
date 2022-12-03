package cache

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestGetCacheLifespan(t *testing.T) {
	now := time.Date(2043, 4, 19, 12, 0, 0, 0, time.UTC)
	timeUntil = func(t time.Time) time.Duration {
		return t.Sub(now)
	}
	tests := []struct {
		headers  http.Header
		expected time.Duration
	}{
		{
			headers: http.Header{
				"Set-Cookie":    {"foobar"},
				"Cache-Control": {"private, max-age=60"},
				"Expires":       {"Sun, 19 Apr 2043 12:00:01 UTC"},
			},
			expected: 0,
		},
		{
			headers: http.Header{
				"Cache-Control": {"private, max-age=60"},
				"Expires":       {"Sun, 19 Apr 2043 12:00:01 UTC"},
			},
			expected: 0,
		},
		{
			headers: http.Header{
				"Cache-Control": {"max-age=60"},
				"Expires":       {"Sun, 19 Apr 2043 12:00:01 UTC"},
			},
			expected: 60 * time.Second,
		},
		{
			headers: http.Header{
				"Cache-Control": {"max-age=0"},
				"Expires":       {"Sun, 19 Apr 2043 12:00:01 UTC"},
			},
			expected: 1 * time.Second,
		},
		{
			headers: http.Header{
				"Expires": {"Sun, 19 Apr 2043 12:00:01 UTC"},
			},
			expected: 1 * time.Second,
		},
		{
			headers:  http.Header{},
			expected: 0,
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("getCacheLifespan, headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expected, getCacheLifespan(test.headers))
		})
	}
}

func TestSetCookieHeaderIsPresent(t *testing.T) {
	tests := []struct {
		headers  http.Header
		expected bool
	}{
		{
			headers:  http.Header{},
			expected: false,
		},
		{
			headers:  http.Header{"Foobar": {}},
			expected: false,
		},
		{
			headers:  http.Header{"Foobar": {"Set-Cookie"}},
			expected: false,
		},
		{
			headers:  http.Header{"Set-Cookie": {}},
			expected: true,
		},
		{
			headers:  http.Header{"Set-Cookie": {"foobar; lorem; ipsum"}},
			expected: true,
		},
		{
			// Header key is case-sensitive
			headers:  http.Header{"set-cookie": {}},
			expected: false,
		},
	}
	mock := &cacheLifespanEvaluator{}
	for _, test := range tests {
		testName := fmt.Sprintf("setCookieHeaderIsPresent, headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			mock.headers = test.headers
			assert.EqualValues(t, test.expected, mock.setCookieHeaderIsPresent())
		})
	}
}

func TestCacheControlHeaderPreventsCaching(t *testing.T) {
	tests := []struct {
		headers  http.Header
		expected bool
	}{
		{
			headers:  http.Header{},
			expected: false,
		},
		{
			headers:  http.Header{"Foobar": {}},
			expected: false,
		},
		{
			headers:  http.Header{"Foobar": {"private"}},
			expected: false,
		},
		{
			headers:  http.Header{"Cache-Control": {}},
			expected: false,
		},
		{
			headers:  http.Header{"Cache-Control": {"foobar"}},
			expected: false,
		},
		{
			headers:  http.Header{"Cache-Control": {"private"}},
			expected: true,
		},
		{
			headers:  http.Header{"Cache-Control": {"no-cache"}},
			expected: true,
		},
		{
			headers:  http.Header{"Cache-Control": {"no-store"}},
			expected: true,
		},
		{
			headers:  http.Header{"Cache-Control": {"private"}},
			expected: true,
		},
		{
			// Header key is case-sensitive
			headers:  http.Header{"cache-control": {"private"}},
			expected: false,
		},
		{
			// Directive is case-insensitive
			headers:  http.Header{"Cache-Control": {"pRivaTe"}},
			expected: true,
		},
	}
	mock := &cacheLifespanEvaluator{}
	for _, test := range tests {
		testName := fmt.Sprintf("cacheControlHeaderPreventsCaching, headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			mock.headers = test.headers
			assert.EqualValues(t, test.expected, mock.cacheControlHeaderPreventsCaching())
		})
	}
}

func TestGetLifespanFromCacheControlHeader(t *testing.T) {
	tests := []struct {
		headers  http.Header
		expected time.Duration
	}{
		{
			headers:  http.Header{},
			expected: 0,
		},
		{
			headers:  http.Header{"Foobar": {}},
			expected: 0,
		},
		{
			headers:  http.Header{"Foobar": {"max-age=60"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Cache-Control": {}},
			expected: 0,
		},
		{
			headers:  http.Header{"Cache-Control": {"foobar"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Cache-Control": {"max-age=0"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Cache-Control": {"max-age=60"}},
			expected: 60 * time.Second,
		},
		{
			// Header key is case-sensitive
			headers:  http.Header{"cache-control": {"max-age=60"}},
			expected: 0,
		},
		{
			// Directive is case-insensitive
			headers:  http.Header{"Cache-Control": {"mAx-AgE=60"}},
			expected: 60 * time.Second,
		},
		{
			headers:  http.Header{"Cache-Control": {"public, max-age=30, max-age=60"}},
			expected: 30 * time.Second,
		},
	}
	mock := &cacheLifespanEvaluator{}
	for _, test := range tests {
		testName := fmt.Sprintf("getLifespanFromCacheControlHeader, headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			mock.headers = test.headers
			assert.EqualValues(t, test.expected, mock.getLifespanFromCacheControlHeader())
		})
	}
}

func TestGetLifespanFromExpiresHeader(t *testing.T) {
	tests := []struct {
		headers  http.Header
		expected time.Duration
	}{
		{
			headers:  http.Header{},
			expected: 0,
		},
		{
			headers:  http.Header{"Foobar": {}},
			expected: 0,
		},
		{
			headers:  http.Header{"Foobar": {"Sun, 19 Apr 2043 12:00:01 UTC"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Expires": {}},
			expected: 0,
		},
		{
			headers:  http.Header{"Expires": {"Invalid timestamp"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Expires": {"Sun, 19 Apr 2043 11:59:59 UTC"}},
			expected: 0,
		},
		{
			headers:  http.Header{"Expires": {"Sun, 19 Apr 2043 12:00:01 UTC"}},
			expected: 1 * time.Second,
		},
		{
			headers:  http.Header{"Expires": {"Invalid", "Sun, 19 Apr 2043 12:00:01 UTC"}},
			expected: 1 * time.Second,
		},
		{
			headers: http.Header{"Expires": {
				"Invalid", "Sun, 19 Apr 2043 12:00:01 UTC", "Sun, 19 Apr 2043 11:59:59 UTC",
			}},
			expected: 1 * time.Second,
		},
	}
	now := time.Date(2043, 4, 19, 12, 0, 0, 0, time.UTC)
	timeUntil = func(t time.Time) time.Duration {
		return t.Sub(now)
	}
	mock := &cacheLifespanEvaluator{}
	for _, test := range tests {
		testName := fmt.Sprintf("getLifespanFromExpiresHeader, headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			mock.headers = test.headers
			assert.EqualValues(t, test.expected, mock.getLifespanFromExpiresHeader())
		})
	}
}

func TestGetDurationUntilTimestamp(t *testing.T) {
	tests := []struct {
		timestamp string
		expected  time.Duration
	}{
		{
			timestamp: "Invalid timestamp",
			expected:  0,
		},
		{
			timestamp: "Sun, 19 Apr 2043 12:00:00 UTC",
			expected:  0,
		},
		{
			timestamp: "Sun, 19 Apr 2043 11:59:59 UTC",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 12:00:01 UTC",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 10:58:59 UTC",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 13:01:01 UTC",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 12:00:00 UTC",
			expected:  0,
		},
		{
			timestamp: "Sunday, 19-Apr-43 11:59:59 UTC",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 12:00:01 UTC",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 10:58:59 UTC",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 13:01:01 UTC",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp: "Sun Apr 19 12:00:00 2043",
			expected:  0,
		},
		{
			timestamp: "Sun Apr 19 11:59:59 2043",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sun Apr 19 12:00:01 2043",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sun Apr 19 10:58:59 2043",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp: "Sun Apr 19 13:01:01 2043",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
	}
	now := time.Date(2043, 4, 19, 12, 0, 0, 0, time.UTC)
	timeUntil = func(t time.Time) time.Duration {
		return t.Sub(now)
	}
	for _, test := range tests {
		testName := fmt.Sprintf(
			"getDurationUntilTimestamp(ts=%s)", test.timestamp)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expected, getDurationUntilTimestamp(test.timestamp))
		})
	}
}

func TestGetDurationSinceTimestamp(t *testing.T) {
	tests := []struct {
		timestamp string
		expected  time.Duration
	}{
		{
			timestamp: "Invalid timestamp",
			expected:  0,
		},
		{
			timestamp: "Sun, 19 Apr 2043 12:00:00 UTC",
			expected:  0,
		},
		{
			timestamp: "Sun, 19 Apr 2043 11:59:59 UTC",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 12:00:01 UTC",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 10:58:59 UTC",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp: "Sun, 19 Apr 2043 13:01:01 UTC",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 12:00:00 UTC",
			expected:  0,
		},
		{
			timestamp: "Sunday, 19-Apr-43 11:59:59 UTC",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 12:00:01 UTC",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 10:58:59 UTC",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp: "Sunday, 19-Apr-43 13:01:01 UTC",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp: "Sun Apr 19 12:00:00 2043",
			expected:  0,
		},
		{
			timestamp: "Sun Apr 19 11:59:59 2043",
			expected:  1 * time.Second,
		},
		{
			timestamp: "Sun Apr 19 12:00:01 2043",
			expected:  -1 * time.Second,
		},
		{
			timestamp: "Sun Apr 19 10:58:59 2043",
			expected:  1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp: "Sun Apr 19 13:01:01 2043",
			expected:  -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
	}
	now := time.Date(2043, 4, 19, 12, 0, 0, 0, time.UTC)
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	for _, test := range tests {
		testName := fmt.Sprintf(
			"getDurationUntilTimestamp(ts=%s)", test.timestamp)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expected, getDurationSinceTimestamp(test.timestamp))
		})
	}
}

func TestGetDurationRelativeToTimestamp(t *testing.T) {
	now := time.Date(2043, 4, 19, 12, 0, 0, 0, time.UTC)
	var timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	var timeUntil = func(t time.Time) time.Duration {
		return t.Sub(now)
	}
	tests := []struct {
		timestamp     string
		timeDeltaFunc func(time.Time) time.Duration
		expected      time.Duration
	}{
		{
			timestamp:     "Invalid timestamp",
			timeDeltaFunc: nil,
			expected:      0,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 12:00:00 UTC",
			timeDeltaFunc: timeSince,
			expected:      0,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 12:00:00 UTC",
			timeDeltaFunc: timeUntil,
			expected:      0,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 11:59:59 UTC",
			timeDeltaFunc: timeSince,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 12:00:01 UTC",
			timeDeltaFunc: timeSince,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 11:59:59 UTC",
			timeDeltaFunc: timeUntil,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 12:00:01 UTC",
			timeDeltaFunc: timeUntil,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 10:58:59 UTC",
			timeDeltaFunc: timeSince,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 13:01:01 UTC",
			timeDeltaFunc: timeSince,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 10:58:59 UTC",
			timeDeltaFunc: timeUntil,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sun, 19 Apr 2043 13:01:01 UTC",
			timeDeltaFunc: timeUntil,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 12:00:00 UTC",
			timeDeltaFunc: timeSince,
			expected:      0,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 12:00:00 UTC",
			timeDeltaFunc: timeUntil,
			expected:      0,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 11:59:59 UTC",
			timeDeltaFunc: timeSince,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 12:00:01 UTC",
			timeDeltaFunc: timeSince,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 11:59:59 UTC",
			timeDeltaFunc: timeUntil,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 12:00:01 UTC",
			timeDeltaFunc: timeUntil,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 10:58:59 UTC",
			timeDeltaFunc: timeSince,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 13:01:01 UTC",
			timeDeltaFunc: timeSince,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 10:58:59 UTC",
			timeDeltaFunc: timeUntil,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sunday, 19-Apr-43 13:01:01 UTC",
			timeDeltaFunc: timeUntil,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},

		{
			timestamp:     "Sun Apr 19 12:00:00 2043",
			timeDeltaFunc: timeSince,
			expected:      0,
		},
		{
			timestamp:     "Sun Apr 19 12:00:00 2043",
			timeDeltaFunc: timeUntil,
			expected:      0,
		},
		{
			timestamp:     "Sun Apr 19 11:59:59 2043",
			timeDeltaFunc: timeSince,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sun Apr 19 12:00:01 2043",
			timeDeltaFunc: timeSince,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sun Apr 19 11:59:59 2043",
			timeDeltaFunc: timeUntil,
			expected:      -1 * time.Second,
		},
		{
			timestamp:     "Sun Apr 19 12:00:01 2043",
			timeDeltaFunc: timeUntil,
			expected:      1 * time.Second,
		},
		{
			timestamp:     "Sun Apr 19 10:58:59 2043",
			timeDeltaFunc: timeSince,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},
		{
			timestamp:     "Sun Apr 19 13:01:01 2043",
			timeDeltaFunc: timeSince,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sun Apr 19 10:58:59 2043",
			timeDeltaFunc: timeUntil,
			expected:      -1*time.Hour - 1*time.Minute - 1*time.Second,
		},
		{
			timestamp:     "Sun Apr 19 13:01:01 2043",
			timeDeltaFunc: timeUntil,
			expected:      1*time.Hour + 1*time.Minute + 1*time.Second,
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf(
			"getDurationRelativeToTimestamp(ts=%s, func=%p)", test.timestamp, test.timeDeltaFunc,
		)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expected, getDurationRelativeToTimestamp(test.timestamp, test.timeDeltaFunc))
		})
	}
}
