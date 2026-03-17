package logwatcher

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	ClientID  string    `json:"client_id,omitempty"`
	Topic     string    `json:"topic,omitempty"`
	Plugin    string    `json:"plugin,omitempty"`
	Raw       string    `json:"raw"`
}

type Filters struct {
	Level    string
	Query    string
	ClientID string
	Topic    string
	From     *time.Time
	To       *time.Time
}

type Parser struct {
	format      string
	customRegex *regexp.Regexp
}

var (
	standardLineRe = regexp.MustCompile(`^(\d+):\s*(.+)$`)
	clientAsRe     = regexp.MustCompile(`(?i)\bas\s+([a-zA-Z0-9._:-]+)\b`)
	clientWordRe   = regexp.MustCompile(`(?i)\bclient\s+([a-zA-Z0-9._:-]+)\b`)
	topicWordRe    = regexp.MustCompile(`(?i)\btopic\s+([^\s]+)\b`)
	topicSubRe     = regexp.MustCompile(`(?i)\bsubscribed\s+to\s+([^\s]+)\b`)
)

func NewParser(format, customRegex string) (*Parser, error) {
	p := &Parser{format: strings.TrimSpace(strings.ToLower(format))}
	if p.format == "" {
		p.format = "mosquitto_standard"
	}
	if customRegex != "" {
		re, err := regexp.Compile(customRegex)
		if err != nil {
			return nil, err
		}
		p.customRegex = re
	}
	if p.format == "custom" && p.customRegex == nil {
		return nil, errors.New("custom format requires custom_regex")
	}
	return p, nil
}

func (p *Parser) ParseLine(line string, id int64) (LogEntry, error) {
	raw := strings.TrimRight(line, "\r\n")
	if raw == "" {
		return LogEntry{}, errors.New("empty line")
	}

	if p.format == "custom" && p.customRegex != nil {
		return p.parseCustom(raw, id)
	}
	return p.parseStandard(raw, id)
}

func (p *Parser) parseStandard(raw string, id int64) (LogEntry, error) {
	m := standardLineRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return LogEntry{}, errors.New("line does not match mosquitto standard format")
	}
	tsUnix, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return LogEntry{}, err
	}
	msg := strings.TrimSpace(m[2])

	entry := LogEntry{
		ID:        id,
		Timestamp: time.Unix(tsUnix, 0).UTC(),
		Level:     detectLevel(msg),
		Message:   msg,
		ClientID:  extractClientID(msg),
		Topic:     extractTopic(msg),
		Raw:       raw,
	}
	return entry, nil
}

func (p *Parser) parseCustom(raw string, id int64) (LogEntry, error) {
	matches := p.customRegex.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return LogEntry{}, errors.New("line does not match custom regex")
	}
	groupMap := map[string]string{}
	for i, name := range p.customRegex.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		groupMap[strings.ToLower(name)] = matches[i]
	}

	msg := strings.TrimSpace(groupMap["msg"])
	tsRaw := strings.TrimSpace(groupMap["ts"])
	level := strings.ToUpper(strings.TrimSpace(groupMap["level"]))
	if level == "" {
		level = detectLevel(msg)
	}

	var ts time.Time
	if tsRaw != "" {
		if i, err := strconv.ParseInt(tsRaw, 10, 64); err == nil {
			ts = time.Unix(i, 0).UTC()
		} else if parsed, err := time.Parse(time.RFC3339, tsRaw); err == nil {
			ts = parsed.UTC()
		}
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	entry := LogEntry{
		ID:        id,
		Timestamp: ts,
		Level:     level,
		Message:   msg,
		ClientID:  firstNonEmpty(strings.TrimSpace(groupMap["client_id"]), extractClientID(msg)),
		Topic:     firstNonEmpty(strings.TrimSpace(groupMap["topic"]), extractTopic(msg)),
		Plugin:    strings.TrimSpace(groupMap["plugin"]),
		Raw:       raw,
	}
	if entry.Message == "" {
		entry.Message = raw
	}
	return entry, nil
}

func detectLevel(message string) string {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "auth failed") || strings.Contains(m, "error") || strings.Contains(m, "failed"):
		return "ERROR"
	case strings.Contains(m, "warning") || strings.Contains(m, "timeout") || strings.Contains(m, "limit exceeded"):
		return "WARN"
	case strings.Contains(m, "mosquitto_sub") || strings.Contains(m, "debug"):
		return "DEBUG"
	default:
		return "INFO"
	}
}

func extractClientID(message string) string {
	if m := clientAsRe.FindStringSubmatch(message); len(m) > 1 {
		return m[1]
	}
	if m := clientWordRe.FindStringSubmatch(message); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractTopic(message string) string {
	if m := topicWordRe.FindStringSubmatch(message); len(m) > 1 {
		return m[1]
	}
	if m := topicSubRe.FindStringSubmatch(message); len(m) > 1 {
		return m[1]
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
