const { createApp, ref, computed, watch } = Vue;

createApp({
  setup() {
    const apiBase = '/api/log';

    // File input
    const filePath = ref('');
    const selectedFile = ref(null);

    // Parse result
    const parseResult = ref(null);
    const parsing = ref(false);
    const loading = ref(false);
    const loadingText = ref('处理中...');

    // Tabs
    const activeTab = ref('frames');
    const tabs = [
      { id: 'frames', name: '帧列表' },
      { id: 'search', name: '搜索' },
      { id: 'analyze', name: '分析' },
      { id: 'export', name: '导出' }
    ];

    // Frames & pagination
    const frames = ref([]);
    const currentPage = ref(1);
    const pageSize = ref(50);
    const totalFrames = ref(0);
    const jumpPage = ref(1);

    // Selected frame
    const selectedFrame = ref(null);
    const frameDetail = ref(null);

    // Frame modal
    const frameModalVisible = ref(false);
    const modalFrame = ref(null);
    const modalFrameDetail = ref(null);

    // Search
    const searchKeyword = ref('');
    const searchResults = ref([]);
    const searching = ref(false);
    const searched = ref(false);
    const searchCurrentPage = ref(1);
    const searchPageSize = ref(20);

    // Analyze
    const topN = ref(20);
    const topFrames = ref([]);
    const shaderStats = ref([]);
    const analyzeLoaded = ref(false);
    const selectedTopFrame = ref(null);
    const selectedTopFrameFuncStats = ref([]);

    // Export
    const exportFormat = ref('json');
    const exporting = ref(false);
    const exportOptions = ref({
      longestFrame: true,
      top20Frames: true,
      shaderLogs: true
    });

    // Toast
    const toast = ref(null);

    // Stop service
    const stopping = ref(false);

    // Computed
    const totalPages = computed(() => Math.max(1, Math.ceil(totalFrames.value / pageSize.value)));

    const pageRange = computed(() => {
      const total = totalPages.value;
      const cur = currentPage.value;
      if (total <= 7) {
        return Array.from({ length: total }, (_, i) => i + 1);
      }
      const range = new Set();
      range.add(1);
      range.add(total);
      range.add(cur);
      range.add(cur - 1);
      range.add(cur + 1);
      if (cur <= 4) {
        range.add(2); range.add(3); range.add(4); range.add(5);
      } else if (cur >= total - 3) {
        range.add(total - 4); range.add(total - 3); range.add(total - 2); range.add(total - 1);
      } else {
        range.add(cur - 2); range.add(cur + 2);
      }
      const sorted = [...range].filter(n => n >= 1 && n <= total).sort((a, b) => a - b);
      const result = [];
      for (let i = 0; i < sorted.length; i++) {
        if (i > 0 && sorted[i] - sorted[i - 1] > 1) result.push('...');
        result.push(sorted[i]);
      }
      return result;
    });

    // Search pagination computed
    const searchTotalPages = computed(() => Math.max(1, Math.ceil(searchResults.value.length / searchPageSize.value)));

    const paginatedSearchResults = computed(() => {
      const start = (searchCurrentPage.value - 1) * searchPageSize.value;
      return searchResults.value.slice(start, start + searchPageSize.value);
    });

    const searchPageRange = computed(() => {
      const total = searchTotalPages.value;
      const cur = searchCurrentPage.value;
      if (total <= 7) {
        return Array.from({ length: total }, (_, i) => i + 1);
      }
      const range = new Set();
      range.add(1);
      range.add(total);
      range.add(cur);
      range.add(cur - 1);
      range.add(cur + 1);
      if (cur <= 4) {
        range.add(2); range.add(3); range.add(4); range.add(5);
      } else if (cur >= total - 3) {
        range.add(total - 4); range.add(total - 3); range.add(total - 2); range.add(total - 1);
      } else {
        range.add(cur - 2); range.add(cur + 2);
      }
      const sorted = [...range].filter(n => n >= 1 && n <= total).sort((a, b) => a - b);
      const result = [];
      for (let i = 0; i < sorted.length; i++) {
        if (i > 0 && sorted[i] - sorted[i - 1] > 1) result.push('...');
        result.push(sorted[i]);
      }
      return result;
    });

    // Toast helper
    function showToast(message, type = 'error') {
      toast.value = { message, type };
      setTimeout(() => { toast.value = null; }, 5000);
    }

    // Stop service
    async function stopService() {
      if (!confirm('确定要停止服务吗？')) return;
      stopping.value = true;
      try {
        const res = await fetch('/api/shutdown', { method: 'POST' });
        if (!res.ok) throw new Error(`停止服务失败: ${res.status}`);
        showToast('服务正在停止...', 'success');
        setTimeout(() => { window.location.reload(); }, 2000);
      } catch (err) {
        showToast(err.message);
      } finally {
        stopping.value = false;
      }
    }

    // File browse
    function browseFile() {
      const input = document.createElement('input');
      input.type = 'file';
      input.accept = '.log,.txt,.trace,.json';
      input.onchange = (e) => {
        const file = e.target.files[0];
        if (file) {
          selectedFile.value = file;
          filePath.value = file.name;
        }
      };
      input.click();
    }

    // Parse
    async function parseFile() {
      if (!selectedFile.value && !filePath.value) return;
      parsing.value = true;
      loading.value = true;
      loadingText.value = '解析日志文件...';

      // AbortController for 120s timeout
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 120000);

      try {
        let res;
        if (selectedFile.value) {
          const formData = new FormData();
          formData.append('file', selectedFile.value);
          formData.append('filename', selectedFile.value.name);
          res = await fetch(`${apiBase}/parse`, {
            method: 'POST',
            body: formData,
            signal: controller.signal
          });
        } else {
          res = await fetch(`${apiBase}/parse`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path: filePath.value }),
            signal: controller.signal
          });
        }

        clearTimeout(timeoutId);
        if (!res.ok) throw new Error(`请求失败: ${res.status}`);
        const data = await res.json();
        parseResult.value = data;
        totalFrames.value = data.frame_count ?? data.total_frames ?? 0;

        await loadFrames();
      } catch (err) {
        clearTimeout(timeoutId);
        if (err.name === 'AbortError') {
          showToast('解析超时（120秒），文件可能过大');
        } else {
          showToast(err.message);
        }
      } finally {
        parsing.value = false;
        loading.value = false;
      }
    }

    // Load frames (paginated)
    async function loadFrames() {
      loading.value = true;
      loadingText.value = '加载帧列表...';
      try {
        const res = await fetch(
          `${apiBase}/frames?page=${currentPage.value}&page_size=${pageSize.value}`
        );
        if (!res.ok) throw new Error(`获取帧列表失败: ${res.status}`);
        const data = await res.json();
        // Handle both array response and {frames: [...], total: ...} response
        let frameArray = [];
        if (Array.isArray(data)) {
          frameArray = data;
        } else if (data && Array.isArray(data.frames)) {
          frameArray = data.frames;
        }
        // Map backend fields to frontend fields
        frames.value = frameArray.map(f => {
          if (!f) return null;
          const frameNum = f.FrameNum ?? f.frame_number ?? f.frameId ?? f.id;
          const totalTimeUs = f.TotalTimeUs ?? f.total_time_us ?? f.duration_us;
          const swapTimeUs = f.SwapBufferTimeUs ?? f.SwapBuffersTimeUs ?? f.swapbuffers_time_us;
          const apiTimeUs = f.APITotalTimeUs ?? f.APITimeUs ?? f.api_time_us;
          const otherTimeUs = f.OtherTimeUs ?? f.other_time_us;
          return {
            id: frameNum,
            duration_us: totalTimeUs,
            duration_ms: totalTimeUs != null ? totalTimeUs / 1000 : null,
            swapbuffers_ms: swapTimeUs != null ? swapTimeUs / 1000 : null,
            api_ms: apiTimeUs != null ? apiTimeUs / 1000 : null,
            other_ms: otherTimeUs != null ? otherTimeUs / 1000 : null,
            api_count: f.APICalls ? f.APICalls.length : (f.api_count ?? 0)
          };
        }).filter(f => f !== null);
        totalFrames.value = data?.total ?? data?.frame_count ?? totalFrames.value;
      } catch (err) {
        showToast(err.message);
      } finally {
        loading.value = false;
      }
    }

    // Pagination
    async function goPage(page) {
      const p = Math.max(1, Math.min(totalPages.value, page));
      currentPage.value = p;
      jumpPage.value = p;
      await loadFrames();
    }

    // Select frame
    async function selectFrame(frame) {
      if (selectedFrame.value && selectedFrame.value.id === frame.id) {
        selectedFrame.value = null;
        frameDetail.value = null;
        return;
      }
      selectedFrame.value = frame;
      try {
        const res = await fetch(`${apiBase}/frames/${frame.id}`);
        if (!res.ok) throw new Error(`获取帧详情失败: ${res.status}`);
        const data = await res.json();
        // Map backend fields to frontend fields
        frameDetail.value = {
          ...data,
          duration_us: data.TotalTimeUs,
          api_count: data.APICalls ? data.APICalls.length : 0,
          // Convert APISummary (map) to api_calls array for display
          api_calls: Object.entries(data.APISummary || {}).map(([name, stats]) => ({
            name: name,
            count: stats.Count,
            total_us: stats.TimeUs
          }))
        };
      } catch (err) {
        showToast(err.message);
      }
    }

    // Frame modal
    async function openFrameModal(frame) {
      modalFrame.value = frame;
      frameModalVisible.value = true;
      try {
        const res = await fetch(`${apiBase}/frames/${frame.id}`);
        if (!res.ok) throw new Error(`获取帧详情失败: ${res.status}`);
        const data = await res.json();
        modalFrameDetail.value = {
          ...data,
          duration_us: data.TotalTimeUs,
          api_count: data.APICalls ? data.APICalls.length : 0,
          api_calls: Object.entries(data.APISummary || {}).map(([name, stats]) => ({
            name: name,
            count: stats.Count,
            duration_us: stats.TimeUs
          }))
        };
      } catch (err) {
        showToast(err.message);
      }
    }

    function closeFrameModal() {
      frameModalVisible.value = false;
      modalFrame.value = null;
      modalFrameDetail.value = null;
    }

    // Search pagination
    function goSearchPage(page) {
      const p = Math.max(1, Math.min(searchTotalPages.value, page));
      searchCurrentPage.value = p;
    }

    // Search
    async function doSearch() {
      if (!searchKeyword.value) return;
      searching.value = true;
      searched.value = false;
      searchResults.value = [];
      searchCurrentPage.value = 1;

      try {
        const res = await fetch(
          `${apiBase}/search?q=${encodeURIComponent(searchKeyword.value)}`
        );
        if (!res.ok) throw new Error(`搜索失败: ${res.status}`);
        const data = await res.json();
        searchResults.value = Array.isArray(data) ? data : (data.results ?? []);
        searched.value = true;
      } catch (err) {
        showToast(err.message);
      } finally {
        searching.value = false;
      }
    }

    // Analyze
    async function loadTopFrames() {
      try {
        // Fixed: always request top 20 frames
        const res = await fetch(`${apiBase}/analyze/top?n=20`);
        if (!res.ok) throw new Error(`获取Top帧失败: ${res.status}`);
        const data = await res.json();
        // Handle both array response and {frames: [...]} response
        let frameArray = [];
        if (Array.isArray(data)) {
          frameArray = data;
        } else if (data && Array.isArray(data.frames)) {
          frameArray = data.frames;
        }
        // Map backend fields to frontend fields
        topFrames.value = frameArray.map(f => {
          if (!f) return null;
          const frameNum = f.FrameNum ?? f.frame_number ?? f.frameId ?? f.id;
          const totalTimeUs = f.TotalTimeUs ?? f.total_time_us ?? f.duration_us;
          const swapTimeUs = f.SwapBufferTimeUs ?? f.SwapBuffersTimeUs ?? f.swapbuffers_time_us;
          const apiTimeUs = f.APITotalTimeUs ?? f.APITimeUs ?? f.api_time_us;
          const otherTimeUs = f.OtherTimeUs ?? f.other_time_us;
          return {
            id: frameNum,
            duration_us: totalTimeUs,
            duration_ms: totalTimeUs != null ? totalTimeUs / 1000 : null,
            swapbuffers_ms: swapTimeUs != null ? swapTimeUs / 1000 : null,
            api_ms: apiTimeUs != null ? apiTimeUs / 1000 : null,
            other_ms: otherTimeUs != null ? otherTimeUs / 1000 : null
          };
        }).filter(f => f !== null);
      } catch (err) {
        showToast(err.message);
        topFrames.value = [];
      }
    }

    // Select top frame
    async function selectTopFrame(frame) {
      if (selectedTopFrame.value && selectedTopFrame.value.id === frame.id) {
        selectedTopFrame.value = null;
        selectedTopFrameFuncStats.value = [];
        return;
      }
      selectedTopFrame.value = frame;
      try {
        const res = await fetch(`${apiBase}/frames/${frame.id}/funcs`);
        if (!res.ok) throw new Error(`获取帧函数统计失败: ${res.status}`);
        const data = await res.json();
        selectedTopFrameFuncStats.value = (Array.isArray(data) ? data : []).map(f => ({
          name: f.FuncName || f.name,
          count: f.CallCount || f.count,
          total_us: f.TotalTimeUs || f.total_us
        }));
      } catch (err) {
        showToast(err.message);
        selectedTopFrameFuncStats.value = [];
      }
    }

    // Toggle shader expand - use index to avoid Vue reactivity issues with mutated objects
    function toggleShaderExpand(shader) {
      const idx = shaderStats.value.findIndex(s => s && (s.id === shader.id || s === shader));
      if (idx !== -1) {
        shaderStats.value[idx].expanded = !shaderStats.value[idx].expanded;
      }
    }

    async function loadShaderStats() {
      try {
        // Note: endpoint is 'shaders' not 'shader'
        const res = await fetch(`${apiBase}/analyze/shaders`);
        if (!res.ok) throw new Error(`获取Shader统计失败: ${res.status}`);
        const data = await res.json();
        // Handle both array response and object with 'shaders' or 'data' property
        let shaderArray = [];
        if (Array.isArray(data)) {
          shaderArray = data;
        } else if (data && Array.isArray(data.shaders)) {
          shaderArray = data.shaders;
        } else if (data && Array.isArray(data.data)) {
          shaderArray = data.data;
        }
        // Map backend fields: ID→id, Source→source, CommandLine→commandLine, Type→name
        shaderStats.value = shaderArray.map(s => {
          if (!s) return null;
          // Truncate very long shader sources to prevent memory issues
          const source = s.Source || s.source || s.Code || s.code || '';
          const truncatedSource = source.length > 50000 ? source.substring(0, 50000) + '...[truncated]' : source;
          return {
            id: s.ID || s.Id || s.id || s.Type || s.ShaderType || 'Unknown',
            name: s.Type || s.ShaderType || s.ID || s.id || 'Unknown',
            source: truncatedSource,
            commandLine: s.CommandLine || s.commandLine || '',
            count: s.CompileCount ?? s.compile_count ?? 0,
            total_us: s.TotalCompileTimeUs ?? s.total_compile_time_us ?? 0,
            expanded: false
          };
        }).filter(s => s !== null);
      } catch (err) {
        showToast(err.message);
        shaderStats.value = [];
      }
    }

    async function loadAnalyze() {
      loading.value = true;
      try {
        await Promise.all([
          loadTopFrames(),
          loadShaderStats()
        ]);
        analyzeLoaded.value = true;
      } catch (err) {
        showToast(err.message);
      } finally {
        loading.value = false;
      }
    }

    // Export
    async function exportData() {
      if (!parseResult.value) return;
      exporting.value = true;
      try {
        const res = await fetch(`${apiBase}/export`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            format: exportFormat.value,
            options: exportOptions.value
          })
        });
        if (!res.ok) throw new Error(`导出失败: ${res.status}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `gpu_log.${exportFormat.value}`;
        a.click();
        URL.revokeObjectURL(url);
        showToast('导出成功', 'success');
      } catch (err) {
        showToast(err.message);
      } finally {
        exporting.value = false;
      }
    }

    // Tab switch
    function switchTab(tabId) {
      activeTab.value = tabId;
      if (tabId === 'analyze' && parseResult.value && !analyzeLoaded.value) {
        loadAnalyze();
      }
    }

    // Reset
    function resetAll() {
      filePath.value = '';
      selectedFile.value = null;
      parseResult.value = null;
      frames.value = [];
      selectedFrame.value = null;
      frameDetail.value = null;
      frameModalVisible.value = false;
      modalFrame.value = null;
      modalFrameDetail.value = null;
      currentPage.value = 1;
      totalFrames.value = 0;
      jumpPage.value = 1;
      searchKeyword.value = '';
      searchResults.value = [];
      searched.value = false;
      searchCurrentPage.value = 1;
      topFrames.value = [];
      shaderStats.value = [];
      analyzeLoaded.value = false;
      selectedTopFrame.value = null;
      selectedTopFrameFuncStats.value = [];
      activeTab.value = 'frames';
      toast.value = null;
    }

    return {
      filePath, selectedFile,
      parseResult, parsing, loading, loadingText,
      activeTab, tabs,
      frames, currentPage, pageSize, totalFrames, totalPages, jumpPage, pageRange,
      selectedFrame, frameDetail,
      frameModalVisible, modalFrame, modalFrameDetail,
      searchKeyword, searchResults, searching, searched,
      searchCurrentPage, searchPageSize, searchTotalPages, paginatedSearchResults, searchPageRange,
      topN, topFrames, shaderStats, selectedTopFrame, selectedTopFrameFuncStats,
      exportFormat, exporting, exportOptions,
      toast, stopping,
      browseFile, parseFile, loadFrames, goPage, selectFrame,
      openFrameModal, closeFrameModal,
      doSearch, goSearchPage,
      loadTopFrames, selectTopFrame, toggleShaderExpand,
      exportData, switchTab, resetAll, stopService
    };
  }
}).mount('#app');
