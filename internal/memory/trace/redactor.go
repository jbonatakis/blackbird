package trace

import (
	"regexp"
	"strings"
)

const DefaultRedactionReplacement = "[redacted]"

var defaultSensitiveHeaders = []string{
	"authorization",
	"proxy-authorization",
	"x-api-key",
	"x-openai-api-key",
	"x-openai-organization",
	"cookie",
	"set-cookie",
	"x-auth-token",
	"x-token",
}

type RedactorConfig struct {
	SensitiveHeaders []string
	ValuePatterns    []string
	Replacement      string
}

type Redactor struct {
	sensitiveHeaders map[string]struct{}
	valuePatterns    []*regexp.Regexp
	replacement      string
}

func DefaultRedactor() *Redactor {
	sensitive := make(map[string]struct{}, len(defaultSensitiveHeaders))
	for _, name := range defaultSensitiveHeaders {
		sensitive[strings.ToLower(name)] = struct{}{}
	}
	return &Redactor{
		sensitiveHeaders: sensitive,
		replacement:      DefaultRedactionReplacement,
	}
}

func NewRedactor(cfg RedactorConfig) (*Redactor, error) {
	sensitive := make(map[string]struct{}, len(cfg.SensitiveHeaders))
	for _, name := range cfg.SensitiveHeaders {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		sensitive[strings.ToLower(trimmed)] = struct{}{}
	}

	patterns := make([]*regexp.Regexp, 0, len(cfg.ValuePatterns))
	for _, pattern := range cfg.ValuePatterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		re, err := regexp.Compile(trimmed)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, re)
	}

	replacement := strings.TrimSpace(cfg.Replacement)
	if replacement == "" {
		replacement = DefaultRedactionReplacement
	}

	return &Redactor{
		sensitiveHeaders: sensitive,
		valuePatterns:    patterns,
		replacement:      replacement,
	}, nil
}

func (r *Redactor) RedactHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}
	if r == nil {
		return cloneHeaders(headers)
	}

	redacted := make(map[string][]string, len(headers))
	for name, values := range headers {
		if len(values) == 0 {
			redacted[name] = nil
			continue
		}

		lower := strings.ToLower(name)
		_, sensitive := r.sensitiveHeaders[lower]
		out := make([]string, len(values))
		for i, value := range values {
			if sensitive || r.matchesValuePattern(value) {
				out[i] = r.replacement
			} else {
				out[i] = value
			}
		}
		redacted[name] = out
	}
	return redacted
}

func (r *Redactor) matchesValuePattern(value string) bool {
	if len(r.valuePatterns) == 0 {
		return false
	}
	for _, re := range r.valuePatterns {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

func cloneHeaders(headers map[string][]string) map[string][]string {
	out := make(map[string][]string, len(headers))
	for name, values := range headers {
		if values == nil {
			out[name] = nil
			continue
		}
		copyValues := make([]string, len(values))
		copy(copyValues, values)
		out[name] = copyValues
	}
	return out
}
