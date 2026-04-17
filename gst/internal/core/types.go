package core

type APILogEntry struct {
	APIName  string
	Count    int
	TimeUs   int64
	LineNum  int
}

type FrameInfo struct {
	FrameNum     int
	StartLine    int
	EndLine      int
	TotalTimeUs  int64
	APICalls     []APILogEntry
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
	Type             string // "Vertex", "Fragment", etc.
	CompileCount     int
	TotalCompileTimeUs int64
}
