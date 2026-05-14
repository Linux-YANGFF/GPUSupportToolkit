import { ref, computed } from 'vue'
import type {
  BottleneckAnalysis,
  DrawCallSummary,
  DrawCallInsight,
  FrameDrawCallPage,
  FrameData,
  FrameDetail,
  FrameProgramInsight,
  FrameRawLinesPage,
  FrameSummaryResponse,
  OverviewResult,
  ParseResult,
  ProgramInfo,
  SearchResponse,
  SearchResultItem,
  ShaderStat,
  TabItem,
  TextureSummary,
  ToastMessage,
  TraceProgramsResponse,
} from '../types'

const FETCH_TIMEOUT = 120000
const TOAST_TIMEOUT = 5000
const apiBase = '/api/log'

export function useLogAnalysis() {
  const filePath = ref('')
  const selectedFile = ref<File | null>(null)

  const parseResult = ref<ParseResult | null>(null)
  const parsing = ref(false)
  const loading = ref(false)
  const loadingText = ref('处理中...')

  const activeTab = ref('frames')
  const tabs: TabItem[] = [
    { id: 'frames', name: '帧列表' },
    { id: 'search', name: '搜索' },
    { id: 'analyze', name: '分析' },
    { id: 'trace', name: 'Trace Inspector' },
    { id: 'export', name: '导出' },
  ]

  const frames = ref<FrameData[]>([])
  const currentPage = ref(1)
  const pageSize = ref(50)
  const totalFrames = ref(0)
  const jumpPage = ref(1)

  const selectedFrame = ref<FrameData | null>(null)
  const frameDetail = ref<FrameDetail | null>(null)

  const frameModalVisible = ref(false)
  const modalFrame = ref<FrameData | null>(null)
  const modalFrameDetail = ref<FrameDetail | null>(null)
  const modalFrameRawLines = ref<string[]>([])
  const modalFrameRawPage = ref(1)
  const modalFrameRawPageSize = ref(200)
  const modalFrameRawTotal = ref(0)
  const modalFrameRawLoading = ref(false)
  const modalFrameRawStartLine = ref(0)
  const modalFrameRawEndLine = ref(0)
  const frameLogDownloading = ref(false)

  const searchKeyword = ref('')
  const searchResults = ref<SearchResultItem[]>([])
  const searchTotal = ref(0)
  const searching = ref(false)
  const searched = ref(false)
  const searchCurrentPage = ref(1)
  const searchPageSize = ref(20)

  const topN = ref(20)
  const topFrames = ref<FrameData[]>([])
  const shaderStats = ref<ShaderStat[]>([])
  const analyzeLoaded = ref(false)
  const analyzeLoading = ref(false)
  const selectedTopFrame = ref<FrameData | null>(null)

  const overview = ref<OverviewResult | null>(null)
  const bottleneck = ref<BottleneckAnalysis | null>(null)
  const drawCallSummary = ref<DrawCallSummary | null>(null)
  const textureSummary = ref<TextureSummary | null>(null)
  const tracePrograms = ref<ProgramInfo[]>([])
  const traceProgramsLoaded = ref(false)
  const traceLoading = ref(false)
  const selectedTraceFrame = ref<FrameData | null>(null)
  const traceFrameJump = ref<number | null>(null)
  const frameProgramInsight = ref<FrameProgramInsight | null>(null)
  const traceDrawCalls = ref<DrawCallInsight[]>([])
  const traceDrawCallPage = ref(1)
  const traceDrawCallPageSize = ref(100)
  const traceDrawCallTotal = ref(0)
  const selectedTraceProgramId = ref<number | null>(null)
  const selectedTraceProgram = ref<ProgramInfo | null>(null)
  const programDetailLoading = ref(false)

  const exportFormat = ref('json')
  const exporting = ref(false)
  const exportType = ref('frames')

  const toast = ref<ToastMessage | null>(null)
  const stopping = ref(false)

  const totalPages = computed(() => Math.max(1, Math.ceil(totalFrames.value / pageSize.value)))

  const pageRange = computed(() => buildPageRange(currentPage.value, totalPages.value))

  const searchTotalPages = computed(() => Math.max(1, Math.ceil(searchTotal.value / searchPageSize.value)))
  const paginatedSearchResults = computed(() => searchResults.value)

  const searchPageRange = computed(() => buildPageRange(searchCurrentPage.value, searchTotalPages.value))

  const hotDrawFrames = computed(() => {
    const all = drawCallSummary.value?.frames ?? []
    return [...all].sort((a, b) => b.total_draw_calls - a.total_draw_calls).slice(0, 10)
  })

  const traceDrawCallTotalPages = computed(() => Math.max(1, Math.ceil(traceDrawCallTotal.value / traceDrawCallPageSize.value)))

  const traceProgramSummary = computed(() => {
    const totalDraws = tracePrograms.value.reduce((sum, p) => sum + (p.draw_call_count || 0), 0)
    const sourcePrograms = tracePrograms.value.filter(p => p.source_type === 'source').length
    const binaryPrograms = tracePrograms.value.filter(p => p.source_type === 'program_binary').length
    return {
      total: tracePrograms.value.length,
      totalDraws,
      sourcePrograms,
      binaryPrograms,
    }
  })

  function showToast(message: string, type: 'error' | 'success' = 'error') {
    toast.value = { message, type }
    setTimeout(() => { toast.value = null }, TOAST_TIMEOUT)
  }

  async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
    const res = await fetch(url, init)
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `请求失败: ${res.status}`)
    }
    return await res.json() as T
  }

  async function stopService() {
    if (!confirm('确定要停止服务吗？')) return
    stopping.value = true
    try {
      const res = await fetch('/api/shutdown', { method: 'POST' })
      if (!res.ok) throw new Error(`停止服务失败: ${res.status}`)
      showToast('服务正在停止...', 'success')
      setTimeout(() => { window.location.reload() }, 2000)
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      stopping.value = false
    }
  }

  function browseFile() {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.log,.txt,.trace,.json'
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file) {
        selectedFile.value = file
        filePath.value = file.name
        if (file.size > 100 * 1024 * 1024) {
          showToast('大于 100MB 的日志建议输入服务器本机路径解析，避免浏览器上传占用过高。')
        }
      }
    }
    input.click()
  }

  async function parseFile() {
    if (!selectedFile.value && !filePath.value) return
    if (selectedFile.value && selectedFile.value.size > 100 * 1024 * 1024) {
      showToast('当前浏览器上传大小超过 100MB，建议改用本机路径解析大日志。')
      return
    }
    parsing.value = true
    loading.value = true
    loadingText.value = '解析日志文件...'

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT)

    try {
      let res: Response
      if (selectedFile.value) {
        const formData = new FormData()
        formData.append('file', selectedFile.value)
        formData.append('filename', selectedFile.value.name)
        res = await fetch(`${apiBase}/parse`, {
          method: 'POST',
          body: formData,
          signal: controller.signal,
        })
      } else {
        res = await fetch(`${apiBase}/parse`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: filePath.value }),
          signal: controller.signal,
        })
      }

      clearTimeout(timeoutId)
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `请求失败: ${res.status}`)
      }
      parseResult.value = await res.json() as ParseResult
      totalFrames.value = parseResult.value.frame_count
      resetAnalysisData()
      await loadFrames()
    } catch (err) {
      clearTimeout(timeoutId)
      if (err instanceof DOMException && err.name === 'AbortError') {
        showToast('解析超时（120秒），文件可能过大')
      } else {
        showToast(err instanceof Error ? err.message : String(err))
      }
    } finally {
      parsing.value = false
      loading.value = false
    }
  }

  async function loadFrames() {
    loading.value = true
    loadingText.value = '加载帧列表...'
    try {
      const data = await fetchJSON<{ frames: FrameSummaryResponse[]; total: number }>(
        `${apiBase}/frames?page=${currentPage.value}&page_size=${pageSize.value}`
      )
      frames.value = data.frames.map(mapFrameData)
      totalFrames.value = data.total
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      loading.value = false
    }
  }

  async function goPage(page: number) {
    const p = Math.max(1, Math.min(totalPages.value, page))
    currentPage.value = p
    jumpPage.value = p
    await loadFrames()
  }

  async function selectFrame(frame: FrameData) {
    if (selectedFrame.value && selectedFrame.value.id === frame.id) {
      selectedFrame.value = null
      frameDetail.value = null
      return
    }
    selectedFrame.value = frame
    try {
      frameDetail.value = await fetchJSON<FrameDetail>(`${apiBase}/frames/${frame.id}`)
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    }
  }

  async function openFrameModal(frame: FrameData) {
    modalFrame.value = frame
    modalFrameDetail.value = null
    modalFrameRawLines.value = []
    modalFrameRawPage.value = 1
    modalFrameRawTotal.value = 0
    modalFrameRawStartLine.value = 0
    modalFrameRawEndLine.value = 0
    frameModalVisible.value = true
    try {
      modalFrameDetail.value = await fetchJSON<FrameDetail>(`${apiBase}/frames/${frame.id}`)
      await loadModalFrameRawLines(1)
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    }
  }

  async function loadModalFrameRawLines(page = modalFrameRawPage.value) {
    if (!modalFrame.value) return
    modalFrameRawLoading.value = true
    try {
      const safePage = Math.max(1, page)
      const data = await fetchJSON<FrameRawLinesPage>(
        `${apiBase}/frames/${modalFrame.value.id}/raw-lines?page=${safePage}&page_size=${modalFrameRawPageSize.value}`
      )
      modalFrameRawLines.value = data.lines
      modalFrameRawPage.value = data.page
      modalFrameRawPageSize.value = data.page_size
      modalFrameRawTotal.value = data.total
      modalFrameRawStartLine.value = data.start_line
      modalFrameRawEndLine.value = data.end_line
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      modalFrameRawLoading.value = false
    }
  }

  async function downloadFrameLog(frame: FrameData | null = modalFrame.value) {
    if (!frame) return
    frameLogDownloading.value = true
    try {
      const res = await fetch(`${apiBase}/frames/${frame.id}/download`)
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `下载失败: ${res.status}`)
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `frame_${frame.id}.log`
      a.click()
      URL.revokeObjectURL(url)
      showToast(`Frame #${frame.id} 日志已下载`, 'success')
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      frameLogDownloading.value = false
    }
  }

  function closeFrameModal() {
    frameModalVisible.value = false
    modalFrame.value = null
    modalFrameDetail.value = null
    modalFrameRawLines.value = []
    modalFrameRawTotal.value = 0
  }

  async function goSearchPage(page: number) {
    const p = Math.max(1, Math.min(searchTotalPages.value, page))
    await loadSearchPage(p)
  }

  async function doSearch() {
    if (!searchKeyword.value) return
    await loadSearchPage(1)
  }

  async function loadSearchPage(page: number) {
    if (!searchKeyword.value) return
    searching.value = true
    searched.value = false
    searchResults.value = []
    searchCurrentPage.value = Math.max(1, page)

    try {
      const data = await fetchJSON<SearchResponse>(
        `${apiBase}/search?q=${encodeURIComponent(searchKeyword.value)}&page=${searchCurrentPage.value}&page_size=${searchPageSize.value}`
      )
      searchResults.value = data.results
      searchTotal.value = data.total
      searchCurrentPage.value = data.page
      searchPageSize.value = data.page_size
      searched.value = true
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      searching.value = false
    }
  }

  async function loadOverview() {
    overview.value = await fetchJSON<OverviewResult>('/api/overview')
  }

  async function loadBottleneck() {
    bottleneck.value = await fetchJSON<BottleneckAnalysis>(`${apiBase}/analyze/bottleneck`)
  }

  async function loadDrawCalls() {
    drawCallSummary.value = await fetchJSON<DrawCallSummary>(`${apiBase}/analyze/drawcalls`)
  }

  async function loadTextures() {
    textureSummary.value = await fetchJSON<TextureSummary>(`${apiBase}/analyze/textures`)
  }

  async function loadTopFrames() {
    const data = await fetchJSON<{ frames: FrameSummaryResponse[] }>(`${apiBase}/analyze/top?n=${topN.value}`)
    topFrames.value = data.frames.map(mapFrameData)
  }

  async function selectTopFrame(frame: FrameData) {
    selectedTopFrame.value = frame
    await openFrameModal(frame)
  }

  function toggleShaderExpand(shader: ShaderStat) {
    const idx = shaderStats.value.findIndex(s => s.id === shader.id)
    if (idx !== -1) {
      shaderStats.value[idx].expanded = !shaderStats.value[idx].expanded
    }
  }

  async function loadShaderStats() {
    const data = await fetchJSON<{ shaders: Omit<ShaderStat, 'expanded'>[] }>(`${apiBase}/analyze/shaders`)
    shaderStats.value = data.shaders.map(shader => ({
      ...shader,
      source: shader.source.length > 50000 ? `${shader.source.substring(0, 50000)}...[truncated]` : shader.source,
      expanded: false,
    }))
  }

  async function loadTracePrograms() {
    if (traceProgramsLoaded.value) return
    traceLoading.value = true
    try {
      const data = await fetchJSON<TraceProgramsResponse>(`${apiBase}/trace/programs`)
      tracePrograms.value = data.programs
      traceProgramsLoaded.value = true
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      traceLoading.value = false
    }
  }

  async function selectTraceFrame(frame: FrameData) {
    selectedTraceFrame.value = frame
    traceFrameJump.value = frame.id
    frameProgramInsight.value = null
    traceDrawCalls.value = []
    traceDrawCallPage.value = 1
    selectedTraceProgramId.value = null
    selectedTraceProgram.value = null
    traceLoading.value = true
    try {
      frameProgramInsight.value = await fetchJSON<FrameProgramInsight>(`${apiBase}/frames/${frame.id}/programs`)
      await loadTraceDrawCalls()
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      traceLoading.value = false
    }
  }

  async function selectTraceFrameById(frameId: number | null) {
    if (frameId == null || Number.isNaN(frameId)) return
    const localFrame = frames.value.find(f => f.id === frameId)
    if (localFrame) {
      await selectTraceFrame(localFrame)
      return
    }
    traceLoading.value = true
    try {
      const detail = await fetchJSON<FrameDetail>(`${apiBase}/frames/${frameId}`)
      await selectTraceFrame(mapFrameData({
        frame_num: detail.frame_num,
        start_line: detail.start_line,
        end_line: detail.end_line,
        total_time_us: detail.total_time_us,
        swap_buffer_time_us: detail.swap_buffer_time_us,
        api_total_time_us: detail.api_total_time_us,
        api_count: detail.api_count,
        draw_call_count: detail.draw_call_count,
        has_timing: detail.has_timing,
        timing_source: detail.timing_source,
      }))
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      traceLoading.value = false
    }
  }

  async function loadTraceDrawCalls(page = traceDrawCallPage.value) {
    if (!selectedTraceFrame.value) return
    traceDrawCallPage.value = Math.max(1, Math.min(traceDrawCallTotalPages.value, page))
    const programQuery = selectedTraceProgramId.value != null ? `&program=${selectedTraceProgramId.value}` : ''
    const data = await fetchJSON<FrameDrawCallPage>(
      `${apiBase}/frames/${selectedTraceFrame.value.id}/drawcalls?page=${traceDrawCallPage.value}&page_size=${traceDrawCallPageSize.value}${programQuery}`
    )
    traceDrawCalls.value = data.draw_calls
    traceDrawCallTotal.value = data.total
    traceDrawCallPage.value = data.page
    traceDrawCallPageSize.value = data.page_size
  }

  async function filterTraceProgram(programId: number | null) {
    selectedTraceProgramId.value = selectedTraceProgramId.value === programId ? null : programId
    traceDrawCallPage.value = 1
    if (programId != null) {
      await loadTraceProgramDetail(programId)
    } else {
      selectedTraceProgram.value = null
    }
    try {
      await loadTraceDrawCalls(1)
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    }
  }

  async function loadTraceProgramDetail(programId: number) {
    programDetailLoading.value = true
    try {
      selectedTraceProgram.value = await fetchJSON<ProgramInfo>(`${apiBase}/trace/programs/${programId}`)
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      programDetailLoading.value = false
    }
  }

  async function openTraceInspector() {
    if (!parseResult.value) return
    await loadTracePrograms()
    if (!selectedTraceFrame.value && frames.value.length > 0) {
      await selectTraceFrame(frames.value[0])
    }
  }

  async function loadAnalyze() {
    if (!parseResult.value || analyzeLoading.value) return
    analyzeLoading.value = true
    loading.value = true
    loadingText.value = '加载分析数据...'
    try {
      await Promise.all([
        loadOverview(),
        loadBottleneck(),
        loadDrawCalls(),
        loadTextures(),
        loadTopFrames(),
        loadShaderStats(),
      ])
      analyzeLoaded.value = true
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      analyzeLoading.value = false
      loading.value = false
    }
  }

  async function exportData() {
    if (!parseResult.value) return
    exporting.value = true
    try {
      const res = await fetch(`${apiBase}/export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          format: exportFormat.value,
          type: exportType.value,
        }),
      })
      if (!res.ok) throw new Error(`导出失败: ${res.status}`)
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `gpu_log.${exportFormat.value}`
      a.click()
      URL.revokeObjectURL(url)
      showToast('导出成功', 'success')
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err))
    } finally {
      exporting.value = false
    }
  }

  function switchTab(tabId: string) {
    activeTab.value = tabId
    if (tabId === 'analyze' && parseResult.value && !analyzeLoaded.value) {
      loadAnalyze()
    }
    if (tabId === 'trace' && parseResult.value) {
      openTraceInspector()
    }
  }

  function resetAnalysisData() {
    topFrames.value = []
    shaderStats.value = []
    analyzeLoaded.value = false
    selectedTopFrame.value = null
    overview.value = null
    bottleneck.value = null
    drawCallSummary.value = null
    textureSummary.value = null
    tracePrograms.value = []
    traceProgramsLoaded.value = false
    traceLoading.value = false
    selectedTraceFrame.value = null
    traceFrameJump.value = null
    frameProgramInsight.value = null
    traceDrawCalls.value = []
    traceDrawCallPage.value = 1
    traceDrawCallTotal.value = 0
    selectedTraceProgramId.value = null
    selectedTraceProgram.value = null
    programDetailLoading.value = false
  }

  function resetAll() {
    filePath.value = ''
    selectedFile.value = null
    parseResult.value = null
    frames.value = []
    selectedFrame.value = null
    frameDetail.value = null
    frameModalVisible.value = false
    modalFrame.value = null
    modalFrameDetail.value = null
    modalFrameRawLines.value = []
    modalFrameRawTotal.value = 0
    currentPage.value = 1
    totalFrames.value = 0
    jumpPage.value = 1
    searchKeyword.value = ''
    searchResults.value = []
    searchTotal.value = 0
    searched.value = false
    searchCurrentPage.value = 1
    resetAnalysisData()
    activeTab.value = 'frames'
    toast.value = null
  }

  return {
    filePath, selectedFile,
    parseResult, parsing, loading, loadingText,
    activeTab, tabs,
    frames, currentPage, pageSize, totalFrames, totalPages, jumpPage, pageRange,
    selectedFrame, frameDetail,
    frameModalVisible, modalFrame, modalFrameDetail,
    modalFrameRawLines, modalFrameRawPage, modalFrameRawPageSize, modalFrameRawTotal,
    modalFrameRawLoading, modalFrameRawStartLine, modalFrameRawEndLine, frameLogDownloading,
    searchKeyword, searchResults, searching, searched,
    searchCurrentPage, searchPageSize, searchTotal, searchTotalPages, paginatedSearchResults, searchPageRange,
    topN, topFrames, shaderStats, selectedTopFrame,
    overview, bottleneck, drawCallSummary, textureSummary, hotDrawFrames,
    tracePrograms, traceProgramsLoaded, traceLoading, traceProgramSummary,
    selectedTraceFrame, frameProgramInsight, traceDrawCalls,
    traceFrameJump,
    traceDrawCallPage, traceDrawCallPageSize, traceDrawCallTotal, traceDrawCallTotalPages,
    selectedTraceProgramId, selectedTraceProgram, programDetailLoading,
    exportFormat, exporting, exportType,
    toast, stopping, analyzeLoading,
    browseFile, parseFile, loadFrames, goPage, selectFrame,
    openFrameModal, loadModalFrameRawLines, downloadFrameLog, closeFrameModal,
    doSearch, goSearchPage,
    loadAnalyze, loadTopFrames, selectTopFrame, toggleShaderExpand,
    loadTracePrograms, selectTraceFrame, selectTraceFrameById, loadTraceDrawCalls, filterTraceProgram, loadTraceProgramDetail,
    exportData, switchTab, resetAll, stopService,
  }
}

function mapFrameData(f: FrameSummaryResponse): FrameData {
  const totalTimeUs = f.total_time_us ?? 0
  const swapTimeUs = f.swap_buffer_time_us ?? 0
  const apiTimeUs = f.api_total_time_us ?? 0
  const otherTimeUs = Math.max(0, totalTimeUs - swapTimeUs - apiTimeUs)
  return {
    id: f.frame_num,
    start_line: f.start_line,
    end_line: f.end_line,
    duration_us: totalTimeUs,
    duration_ms: totalTimeUs / 1000,
    swap_buffer_time_us: swapTimeUs,
    swapbuffers_ms: swapTimeUs > 0 ? swapTimeUs / 1000 : null,
    api_total_time_us: apiTimeUs,
    api_ms: apiTimeUs > 0 ? apiTimeUs / 1000 : null,
    other_ms: otherTimeUs > 0 ? otherTimeUs / 1000 : null,
    api_count: f.api_count ?? 0,
    draw_call_count: f.draw_call_count ?? 0,
    has_timing: Boolean(f.has_timing ?? totalTimeUs > 0),
    timing_source: f.timing_source || (totalTimeUs > 0 ? 'profile' : 'none'),
    stats_source: f.stats_source,
    category_stats: f.category_stats ?? [],
    key_apis: f.key_apis ?? [],
  }
}

function buildPageRange(current: number, total: number): (number | string)[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const range = new Set<number>()
  range.add(1)
  range.add(total)
  range.add(current)
  range.add(current - 1)
  range.add(current + 1)
  if (current <= 4) {
    range.add(2); range.add(3); range.add(4); range.add(5)
  } else if (current >= total - 3) {
    range.add(total - 4); range.add(total - 3); range.add(total - 2); range.add(total - 1)
  } else {
    range.add(current - 2); range.add(current + 2)
  }
  const sorted = [...range].filter(n => n >= 1 && n <= total).sort((a, b) => a - b)
  const result: (number | string)[] = []
  for (let i = 0; i < sorted.length; i++) {
    if (i > 0 && sorted[i] - sorted[i - 1] > 1) result.push('...')
    result.push(sorted[i])
  }
  return result
}
