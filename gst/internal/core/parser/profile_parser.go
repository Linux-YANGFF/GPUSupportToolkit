package parser

import (
	"io"
	"gst/internal/core"
)

// ProfileParser Profile日志解析器
// Profile格式与API格式基本相同，复用APIParser的实现
type ProfileParser struct {
	apiParser *APIParser
}

// NewProfileParser 创建Profile解析器
func NewProfileParser() *ProfileParser {
	return &ProfileParser{
		apiParser: &APIParser{},
	}
}

func (p *ProfileParser) Kind() LogKind { return KindProfile }

func (p *ProfileParser) Parse(reader io.Reader) (*core.ParsedLog, error) {
	// Profile格式与API格式相同，直接复用APIParser
	return p.apiParser.Parse(reader)
}
