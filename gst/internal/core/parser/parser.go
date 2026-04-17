package parser

import (
	"io"
	"strings"
	"gst/internal/core"
)

// LogKind 日志类型
type LogKind string

const (
	KindAPITrace LogKind = "apitrace"
	KindProfile  LogKind = "profile"
	KindUnknown  LogKind = "unknown"
)

// Parser 解析器接口
type Parser interface {
	Parse(reader io.Reader) (*core.ParsedLog, error)
	Kind() LogKind
}

// DetectKind 自动检测日志类型
func DetectKind(firstLine string) LogKind {
	if firstLine == "" {
		return KindUnknown
	}
	// Profile格式通常以 "<<" 开头
	if len(firstLine) >= 2 && firstLine[:2] == "<<" {
		return KindProfile
	}
	// API trace格式包含 "count=" 和 "time="
	if strings.Contains(firstLine, "count=") && strings.Contains(firstLine, "time=") {
		return KindAPITrace
	}
	return KindUnknown
}

// CreateParser 创建对应类型的解析器
func CreateParser(kind LogKind) Parser {
	switch kind {
	case KindAPITrace:
		return &APIParser{}
	case KindProfile:
		return &ProfileParser{}
	default:
		return &APIParser{}
	}
}
