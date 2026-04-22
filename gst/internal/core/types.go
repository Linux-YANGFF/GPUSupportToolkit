package core

type APILogEntry struct {
	APIName   string
	Count     int
	TimeUs    int64
	LineNum   int
	RawParams string // Original parameters for raw trace format
}

type FrameInfo struct {
	FrameNum          int
	StartLine         int
	EndLine           int
	TotalTimeUs       int64
	SwapBufferTimeUs  int64        // swapBuffers 耗时
	APITotalTimeUs    int64        // API 调用总耗时（不含 swapBuffers）
	APICalls          []APILogEntry
	APISummary        map[string]*APISummary
	Shaders           []*ShaderInfo
	Programs          []int        // Program IDs used in this frame (from glUseProgram)
	BufferCreations   []BufferInfo // Buffers created in this frame
}

type BufferInfo struct {
	ID     int
	Target string // e.g., "GL_ARRAY_BUFFER", "GL_ELEMENT_ARRAY_BUFFER"
	Size   int64  // bytes
	Usage  string // e.g., "GL_STATIC_DRAW", "GL_DYNAMIC_DRAW"
}

type ParsedLog struct {
	Frames       []FrameInfo
	TotalTimeUs  int64
	FPS          float64
}

type SearchResult struct {
	LineNum   int
	Content   string
	PageNum   int
}

type FuncStats struct {
	FuncName      string
	CallCount     int
	TotalTimeUs   int64
	AvgTimeUs     int64
}

type ShaderInfo struct {
	ID          int
	CommandLine string `json:"CommandLine,omitempty"` // 原始 glShaderSource 行
	Source      string
}

type APISummary struct {
	APIName string
	Count   int
	TimeUs  int64
}

type ShaderCompileInfo struct {
	Type               string // "Vertex", "Fragment", etc.
	CompileCount       int
	TotalCompileTimeUs int64
}
