export interface FrameData {
  id: number
  start_line?: number
  end_line?: number
  duration_us: number
  duration_ms: number | null
  swap_buffer_time_us?: number
  swapbuffers_ms: number | null
  api_total_time_us?: number
  api_ms: number | null
  other_ms: number | null
  api_count: number
  draw_call_count: number
  has_timing: boolean
  timing_source: string
  stats_source?: string
  category_stats?: CategoryCounter[]
  key_apis?: ApiCounter[]
}

export interface ApiCall {
  name: string
  count: number
  time_us: number
  line_num: number
  raw_params?: string
  gc_addr?: string
  tid?: string
  is_error?: boolean
  error_code?: string
  has_nil_ptr?: boolean
  category?: string
  category_label?: string
  family?: string
  key?: string
  source?: string
}

export interface ApiCounter {
  api_name: string
  category: string
  label: string
  family: string
  key?: string
  count: number
  time_us: number
  avg_time_us: number
  source: string
  has_timing: boolean
  raw_sequence?: number
}

export interface CategoryCounter {
  category: string
  label: string
  count: number
  time_us: number
  avg_time_us: number
  top_apis: ApiCounter[]
}

export interface FrameOpenGLStats {
  frame_num: number
  start_line: number
  end_line: number
  total_time_us: number
  swap_buffer_time_us: number
  api_total_time_us: number
  api_call_count: number
  raw_api_call_count: number
  draw_call_count: number
  has_timing: boolean
  timing_source: string
  stats_source: string
  category_stats: CategoryCounter[]
  key_apis: ApiCounter[]
  top_apis: ApiCounter[]
}

export interface FuncStat {
  name: string
  call_count: number
  total_time_us: number
  avg_time_us: number
}

export interface ShaderStat {
  id: number
  kind?: 'source' | 'api_stat'
  api_name?: string
  count?: number
  time_us?: number
  avg_time_us?: number
  source: string
  command_line?: string
  expanded: boolean
}

export interface FrameDetail {
  frame_num: number
  start_line: number
  end_line: number
  total_time_us: number
  swap_buffer_time_us: number
  api_total_time_us: number
  api_count: number
  draw_call_count: number
  has_timing: boolean
  timing_source: string
  api_calls: ApiCall[]
  func_stats: FuncStat[]
  shaders: ShaderStat[]
  programs: number[]
  stats: FrameOpenGLStats
}

export interface ShaderObjectInfo {
  id: number
  type: string
  create_line?: number
  source_line?: number
  compile_line?: number
  source?: string
  source_available: boolean
  gc_addr?: string
}

export interface ProgramInfo {
  id: number
  source_type: 'source' | 'program_binary' | 'unknown'
  confidence: 'high' | 'medium' | 'low'
  create_line?: number
  link_lines?: number[]
  binary_line?: number
  binary_format?: string
  binary_size_bytes?: number
  shader_ids: number[]
  shaders?: ShaderObjectInfo[]
  frames_used: number[]
  draw_call_count: number
  use_count: number
  first_line?: number
  last_line?: number
}

export interface ProgramUsage {
  program_id: number
  source_type: 'source' | 'program_binary' | 'unknown'
  confidence: 'high' | 'medium' | 'low'
  shader_ids: number[]
  shaders?: ShaderObjectInfo[]
  draw_call_count: number
  use_count: number
  first_line?: number
  last_line?: number
  source_available: boolean
  binary_size_bytes?: number
}

export interface ProgramSegment {
  program_id: number
  start_line: number
  end_line: number
  draw_call_count: number
}

export interface FrameProgramInsight {
  frame_num: number
  start_line: number
  end_line: number
  has_timing: boolean
  total_time_us: number
  api_call_count: number
  total_draw_calls: number
  programs: ProgramUsage[]
  segments: ProgramSegment[]
}

export interface DrawCallInsight {
  index: number
  line_num: number
  api_name: string
  draw_type: string
  raw_params?: string
  program_id: number
  gc_addr?: string
  tid?: string
  vao?: number
  array_buffer?: number
  element_buffer?: number
  fbo?: number
  textures?: Record<string, number>
  confidence: 'high' | 'medium' | 'low'
}

export interface FrameDrawCallPage {
  frame_num: number
  draw_calls: DrawCallInsight[]
  total: number
  page: number
  page_size: number
}

export interface FrameAPICallPage {
  frame_num: number
  api_calls: ApiCall[]
  total: number
  page: number
  page_size: number
}

export interface FrameRawLinesPage {
  frame_num: number
  lines: string[]
  total: number
  page: number
  page_size: number
  start_line: number
  end_line: number
  stripped_line_number: boolean
}

export interface TraceProgramsResponse {
  programs: ProgramInfo[]
  total: number
}

export interface ParseResult {
  format: string
  frame_count: number
  fps: number
  max_frame_time: number
  total_time_us: number
  has_timing: boolean
}

export interface SearchResultItem {
  line_number: number
  content: string
}

export interface SearchResponse {
  results: SearchResultItem[]
  total: number
  page: number
  page_size: number
}

export interface ToastMessage {
  message: string
  type: 'error' | 'success'
}

export interface TabItem {
  id: string
  name: string
}

export interface OverviewResult {
  basic: {
    format: string
    frame_count: number
    total_time_ms: number
    fps: number
  }
  performance: {
    frame_time: {
      min_us: number
      max_us: number
      avg_us: number
      p95_us: number
      p99_us: number
    }
    slow_frames_top5: FrameSummaryResponse[]
    bottleneck_hint: string
  }
  diagnosis_summary: {
    total_findings: number
    critical: number
    high: number
    medium: number
    low: number
    top_issues: string[]
  }
  resources: {
    shader_count: number
    buffer_count: number
    buffer_total_size_bytes: number
    draw_calls_per_frame_avg: number
    total_api_calls: number
  }
  summary: string
}

export interface FrameSummaryResponse {
  frame_num: number
  start_line: number
  end_line: number
  total_time_us: number
  swap_buffer_time_us: number
  api_total_time_us: number
  api_count: number
  draw_call_count: number
  has_timing: boolean
  timing_source: string
  stats_source?: string
  category_stats?: CategoryCounter[]
  key_apis?: ApiCounter[]
}

export interface DrawCallStats {
  frame_num: number
  total_draw_calls: number
  draw_arrays_count: number
  draw_elements_count: number
  instanced_count: number
  indirect_count: number
  compute_count: number
  time_us: number
  has_timing: boolean
}

export interface DrawCallSummary {
  total_draw_calls: number
  draw_calls_per_frame_avg: number
  has_timing: boolean
  by_type: Record<string, number>
  frames: DrawCallStats[]
}

export interface TextureInfo {
  id: number
  target: string
  width?: number
  height?: number
  format?: string
  created: boolean
  bound: boolean
  deleted: boolean
  frame_num?: number
  create_line?: number
  last_bind_line?: number
  source?: string
  bind_count: number
}

export interface TextureSummary {
  total_textures: number
  active_textures: number
  inferred_count: number
  leaked_textures: TextureInfo[]
  by_target: Record<string, number>
}

export interface BottleneckAnalysis {
  type: 'cpu_bound' | 'gpu_bound' | 'balanced' | 'unstable' | 'unknown'
  has_timing: boolean
  confidence: number
  swap_ratio: number
  api_ratio: number
  stability: number
  top_bottleneck: string
  details?: string
}

export interface WorkflowResult {
  workflow: string
  conclusion: string
  evidence: string[]
  details: Record<string, unknown>
}
