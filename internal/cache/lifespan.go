package cache

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

func getCacheLifespan(headers http.Header) time.Duration {
	evaluator := cacheLifespanEvaluator{
		headers: headers,
	}
	if evaluator.setCookieHeaderIsPresent() || evaluator.cacheControlHeaderPreventsCaching() {
		return 0
	}
	if lifespan := evaluator.getLifespanFromCacheControlHeader(); lifespan > 0 {
		return lifespan
	}
	return evaluator.getLifespanFromExpiresHeader()
}

type cacheLifespanEvaluator struct {
	headers http.Header
}

func (evaluator *cacheLifespanEvaluator) setCookieHeaderIsPresent() bool {
	_, ok := evaluator.headers["Set-Cookie"]
	return ok
}

func (evaluator *cacheLifespanEvaluator) cacheControlHeaderPreventsCaching() bool {
	for _, value := range evaluator.headers["Cache-Control"] {
		if regexp.MustCompile(`(?i)private|no-cache|no-store`).FindString(value) != "" {
			return true
		}
	}
	return false
}

func (evaluator *cacheLifespanEvaluator) getLifespanFromCacheControlHeader() time.Duration {
	for _, value := range evaluator.headers["Cache-Control"] {
		maxAgeDirective := regexp.MustCompile(`(?i)max-age=\d+`).FindString(value)
		if maxAgeDirective == "" {
			continue
		}
		maxAgeAsString := strings.ReplaceAll(strings.ToLower(maxAgeDirective), "max-age=", "")
		maxAge, _ := time.ParseDuration(maxAgeAsString + "s") // s for "seconds"
		if maxAge > 0 {
			return maxAge
		}
	}
	return 0
}

func (evaluator *cacheLifespanEvaluator) getLifespanFromExpiresHeader() time.Duration {
	for _, value := range evaluator.headers["Expires"] {
		if lifespan := getLifespanFromStringValue(value); lifespan > 0 {
			return lifespan
		}
	}
	return 0
}

var getLifespanFromStringValue = func() func(string) time.Duration {
	httpTimestampFormats := []string{time.RFC1123, time.RFC850, time.ANSIC}
	return func(value string) time.Duration {
		// See RFC 7231, section 7.1.1.1
		// https://datatracker.ietf.org/doc/html/rfc7231#section-7.1.1.1
		for _, layout := range httpTimestampFormats {
			if expireTime, err := time.Parse(layout, value); err == nil {
				return time.Until(expireTime)
			}
		}
		return 0
	}
}()
