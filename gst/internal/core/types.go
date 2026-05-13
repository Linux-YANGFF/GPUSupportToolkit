package core

type APILogEntry struct {
	APIName     string
	Count       int
	TimeUs      int64
	LineNum     int
	RawParams   string // Original parameters for raw trace format
	ReturnValue string `json:"return_value,omitempty"`
	GCAddr      string `json:"gc_addr,omitempty"`
	TID         string `json:"tid,omitempty"`
	IsError     bool   `json:"is_error,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	HasNilPtr   bool   `json:"has_nil_ptr,omitempty"`
}

type FrameInfo struct {
	FrameNum         int
	StartLine        int
	EndLine          int
	TotalTimeUs      int64
	SwapBufferTimeUs int64 // swapBuffers 耗时
	APITotalTimeUs   int64 // API 调用总耗时（不含 swapBuffers）
	APICalls         []APILogEntry
	APISummary       map[string]*APISummary
	Shaders          []*ShaderInfo
	Programs         []int        // Program IDs used in this frame (from glUseProgram)
	BufferCreations  []BufferInfo // Buffers created in this frame
}

type BufferInfo struct {
	ID     int
	Target string // e.g., "GL_ARRAY_BUFFER", "GL_ELEMENT_ARRAY_BUFFER"
	Size   int64  // bytes
	Usage  string // e.g., "GL_STATIC_DRAW", "GL_DYNAMIC_DRAW"
}

type ParsedLog struct {
	Frames      []FrameInfo
	TotalTimeUs int64
	FPS         float64
}

type SearchResult struct {
	LineNum int
	Content string
	PageNum int
}

type FuncStats struct {
	FuncName    string
	CallCount   int
	TotalTimeUs int64
	AvgTimeUs   int64
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

type FrameSummary struct {
	TotalFrames int   `json:"total_frames"`
	AvgTimeUs   int64 `json:"avg_time_us"`
	MaxTimeUs   int64 `json:"max_time_us"`
	MinTimeUs   int64 `json:"min_time_us"`
	TotalTimeUs int64 `json:"total_time_us"`
}

type BufferTargetStat struct {
	Count     int   `json:"count"`
	TotalSize int64 `json:"total_size"`
}

type BufferSummary struct {
	TotalCount  int                         `json:"total_count"`
	TotalSize   int64                       `json:"total_size"`
	TargetStats map[string]BufferTargetStat `json:"target_stats"`
}

type FuncSummary struct {
	TotalFunctions int         `json:"total_functions"`
	TotalCalls     int         `json:"total_calls"`
	TotalTimeUs    int64       `json:"total_time_us"`
	TopFunctions   []FuncStats `json:"top_functions"`
}

type ShaderSummary struct {
	ShaderTypes  int   `json:"shader_types"`
	TotalCompile int   `json:"total_compile"`
	TotalTimeUs  int64 `json:"total_time_us"`
}

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

type FindingConfidence string

const (
	ConfidenceHigh   FindingConfidence = "high"
	ConfidenceMedium FindingConfidence = "medium"
	ConfidenceLow    FindingConfidence = "low"
)

type FindingKind string

const (
	FindingKindBug           FindingKind = "bug"
	FindingKindPerformance   FindingKind = "performance"
	FindingKindCompatibility FindingKind = "compatibility"
	FindingKindInfo          FindingKind = "info"
)

type Finding struct {
	Severity       Severity          `json:"severity"`
	Category       string            `json:"category,omitempty"`
	Kind           FindingKind       `json:"kind,omitempty"`
	Confidence     FindingConfidence `json:"confidence,omitempty"`
	Count          int               `json:"count,omitempty"`
	Examples       []string          `json:"examples,omitempty"`
	Description    string            `json:"description"`
	Evidence       string            `json:"evidence"`
	RootCauseChain []string          `json:"root_cause_chain"`
	FixSuggestion  string            `json:"fix_suggestion"`
}

type DiagnosisSummary struct {
	TotalFindings int `json:"total_findings"`
	CriticalCount int `json:"critical_count"`
	HighCount     int `json:"high_count"`
	MediumCount   int `json:"medium_count"`
	LowCount      int `json:"low_count"`
	InfoCount     int `json:"info_count"`
}

type DiagnosisReport struct {
	SourceFile  string           `json:"source_file"`
	GeneratedAt string           `json:"generated_at"`
	Summary     DiagnosisSummary `json:"summary"`
	Findings    []Finding        `json:"findings"`
}

// Overview types

type OverviewBasic struct {
	Format      string  `json:"format"`
	FrameCount  int     `json:"frame_count"`
	TotalTimeMs int64   `json:"total_time_ms"`
	FPS         float64 `json:"fps"`
}

type OverviewFrameTime struct {
	MinUs int64 `json:"min_us"`
	MaxUs int64 `json:"max_us"`
	AvgUs int64 `json:"avg_us"`
	P95Us int64 `json:"p95_us"`
	P99Us int64 `json:"p99_us"`
}

type OverviewPerformance struct {
	FrameTime      OverviewFrameTime `json:"frame_time"`
	SlowFramesTop5 []FrameInfo       `json:"slow_frames_top5"`
	BottleneckHint string            `json:"bottleneck_hint"`
}

type OverviewDiagnosis struct {
	TotalFindings int      `json:"total_findings"`
	Critical      int      `json:"critical"`
	High          int      `json:"high"`
	Medium        int      `json:"medium"`
	Low           int      `json:"low"`
	TopIssues     []string `json:"top_issues"`
}

type OverviewResources struct {
	ShaderCount          int   `json:"shader_count"`
	BufferCount          int   `json:"buffer_count"`
	BufferTotalSizeBytes int64 `json:"buffer_total_size_bytes"`
	DrawCallsPerFrameAvg int   `json:"draw_calls_per_frame_avg"`
	TotalAPICalls        int   `json:"total_api_calls"`
}

type OverviewResult struct {
	Basic            OverviewBasic       `json:"basic"`
	Performance      OverviewPerformance `json:"performance"`
	DiagnosisSummary OverviewDiagnosis   `json:"diagnosis_summary"`
	Resources        OverviewResources   `json:"resources"`
	Summary          string              `json:"summary"`
}

// Workflow types

type WorkflowRequest struct {
	Workflow string `json:"workflow"`
}

type WorkflowResult struct {
	Workflow   string      `json:"workflow"`
	Conclusion string      `json:"conclusion"`
	Evidence   []string    `json:"evidence"`
	Details    interface{} `json:"details"`
}

// DrawCall types

type DrawCallStats struct {
	FrameNum          int   `json:"frame_num"`
	TotalDrawCalls    int   `json:"total_draw_calls"`
	DrawArraysCount   int   `json:"draw_arrays_count"`
	DrawElementsCount int   `json:"draw_elements_count"`
	InstancedCount    int   `json:"instanced_count"`
	IndirectCount     int   `json:"indirect_count"`
	ComputeCount      int   `json:"compute_count"`
	TimeUs            int64 `json:"time_us"`
}

type DrawCallSummary struct {
	TotalDrawCalls       int             `json:"total_draw_calls"`
	DrawCallsPerFrameAvg float64         `json:"draw_calls_per_frame_avg"`
	ByType               map[string]int  `json:"by_type"`
	Frames               []DrawCallStats `json:"frames"`
}

// Trace Inspector types

type ShaderObjectInfo struct {
	ID              int    `json:"id"`
	Type            string `json:"type"`
	CreateLine      int    `json:"create_line,omitempty"`
	SourceLine      int    `json:"source_line,omitempty"`
	CompileLine     int    `json:"compile_line,omitempty"`
	Source          string `json:"source,omitempty"`
	SourceAvailable bool   `json:"source_available"`
	GCAddr          string `json:"gc_addr,omitempty"`
}

type ProgramInfo struct {
	ID              int                `json:"id"`
	SourceType      string             `json:"source_type"` // source, program_binary, unknown
	Confidence      string             `json:"confidence"`  // high, medium, low
	CreateLine      int                `json:"create_line,omitempty"`
	LinkLines       []int              `json:"link_lines,omitempty"`
	BinaryLine      int                `json:"binary_line,omitempty"`
	BinaryFormat    string             `json:"binary_format,omitempty"`
	BinarySizeBytes int                `json:"binary_size_bytes,omitempty"`
	ShaderIDs       []int              `json:"shader_ids"`
	Shaders         []ShaderObjectInfo `json:"shaders,omitempty"`
	FramesUsed      []int              `json:"frames_used"`
	DrawCallCount   int                `json:"draw_call_count"`
	UseCount        int                `json:"use_count"`
	FirstLine       int                `json:"first_line,omitempty"`
	LastLine        int                `json:"last_line,omitempty"`
}

type ProgramUsage struct {
	ProgramID       int    `json:"program_id"`
	SourceType      string `json:"source_type"`
	Confidence      string `json:"confidence"`
	ShaderIDs       []int  `json:"shader_ids"`
	DrawCallCount   int    `json:"draw_call_count"`
	UseCount        int    `json:"use_count"`
	FirstLine       int    `json:"first_line,omitempty"`
	LastLine        int    `json:"last_line,omitempty"`
	SourceAvailable bool   `json:"source_available"`
	BinarySizeBytes int    `json:"binary_size_bytes,omitempty"`
}

type ProgramSegment struct {
	ProgramID     int `json:"program_id"`
	StartLine     int `json:"start_line"`
	EndLine       int `json:"end_line"`
	DrawCallCount int `json:"draw_call_count"`
}

type DrawCallInsight struct {
	LineNum       int         `json:"line_num"`
	APIName       string      `json:"api_name"`
	DrawType      string      `json:"draw_type"`
	RawParams     string      `json:"raw_params,omitempty"`
	ProgramID     int         `json:"program_id"`
	GCAddr        string      `json:"gc_addr,omitempty"`
	TID           string      `json:"tid,omitempty"`
	VAO           int         `json:"vao,omitempty"`
	ArrayBuffer   int         `json:"array_buffer,omitempty"`
	ElementBuffer int         `json:"element_buffer,omitempty"`
	FBO           int         `json:"fbo,omitempty"`
	Textures      map[int]int `json:"textures,omitempty"`
	Confidence    string      `json:"confidence"`
}

type FrameProgramInsight struct {
	FrameNum       int              `json:"frame_num"`
	StartLine      int              `json:"start_line"`
	EndLine        int              `json:"end_line"`
	TotalDrawCalls int              `json:"total_draw_calls"`
	Programs       []ProgramUsage   `json:"programs"`
	Segments       []ProgramSegment `json:"segments"`
}

type FrameDrawCallPage struct {
	FrameNum  int               `json:"frame_num"`
	DrawCalls []DrawCallInsight `json:"draw_calls"`
	Total     int               `json:"total"`
	Page      int               `json:"page"`
	PageSize  int               `json:"page_size"`
}

type TraceAnalysis struct {
	Programs      []ProgramInfo                `json:"programs"`
	ProgramMap    map[int]*ProgramInfo         `json:"-"`
	FrameInsights map[int]*FrameProgramInsight `json:"-"`
	DrawCalls     map[int][]DrawCallInsight    `json:"-"`
}

// Texture types

type TextureInfo struct {
	ID       int    `json:"id"`
	Target   string `json:"target"` // e.g., "GL_TEXTURE_2D"
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Format   string `json:"format,omitempty"`
	Created  bool   `json:"created"`
	Bound    bool   `json:"bound"`
	Deleted  bool   `json:"deleted"`
	FrameNum int    `json:"frame_num,omitempty"`
}

type TextureSummary struct {
	TotalTextures  int            `json:"total_textures"`
	ActiveTextures int            `json:"active_textures"`
	LeakedTextures []TextureInfo  `json:"leaked_textures"`
	ByTarget       map[string]int `json:"by_target"`
}

// Bottleneck types

type BottleneckType string

const (
	BottleneckCPU      BottleneckType = "cpu_bound"
	BottleneckGPU      BottleneckType = "gpu_bound"
	BottleneckBalanced BottleneckType = "balanced"
	BottleneckUnstable BottleneckType = "unstable"
)

type BottleneckAnalysis struct {
	Type          BottleneckType `json:"type"`
	Confidence    float64        `json:"confidence"`     // 0.0 - 1.0
	SwapRatio     float64        `json:"swap_ratio"`     // SwapBuffer时间占比
	APIRatio      float64        `json:"api_ratio"`      // API调用时间占比
	Stability     float64        `json:"stability"`      // 帧时间稳定性 (CV)
	TopBottleneck string         `json:"top_bottleneck"` // 最主要的瓶颈函数
	Details       string         `json:"details"`
}
