package search

import (
	"sync"

	"gst/internal/core"
)

// KeywordIndex 关键字索引（倒排索引）
type KeywordIndex struct {
	mu    sync.RWMutex
	index map[string][]int64 // keyword -> line numbers
}

func NewKeywordIndex() *KeywordIndex {
	return &KeywordIndex{
		index: make(map[string][]int64),
	}
}

// Build 构建索引
// 从已解析的日志构建倒排索引，加速后续搜索
func (ki *KeywordIndex) Build(log *core.ParsedLog) {
	ki.mu.Lock()
	defer ki.mu.Unlock()

	ki.index = make(map[string][]int64)

	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			// Index function name
			funcName := call.APIName
			ki.index[funcName] = append(ki.index[funcName], int64(call.LineNum))

			// Index function name prefix (e.g., gl, glut, etc.)
			// Can be extended as needed
		}
	}
}

// Search 搜索关键字
func (ki *KeywordIndex) Search(keyword string) []int64 {
	ki.mu.RLock()
	defer ki.mu.RUnlock()

	return ki.index[keyword]
}

// HasIndexed 检查是否已索引
func (ki *KeywordIndex) HasIndexed() bool {
	ki.mu.RLock()
	defer ki.mu.RUnlock()

	return len(ki.index) > 0
}