<script setup lang="ts">
import { computed, inject } from 'vue'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const {
  parseResult,
  overview,
  bottleneck,
  drawCallSummary,
  textureSummary,
  hotDrawFrames,
  topFrames,
  shaderStats,
  selectedTopFrame,
} = ctx

const { selectTopFrame, toggleShaderExpand, downloadFrameLog } = ctx

const bottleneckLabel = computed(() => {
  const type = bottleneck.value?.type
  if (type === 'cpu_bound') return 'CPU Bound'
  if (type === 'gpu_bound') return 'GPU Bound'
  if (type === 'balanced') return 'Balanced'
  if (type === 'unstable') return 'Unstable'
  if (type === 'unknown') return 'No Timing'
  return '—'
})

function ms(us?: number): string {
  if (us == null) return '—'
  return (us / 1000).toFixed(3)
}

function frameMs(frame: { duration_ms?: number | null; has_timing?: boolean }): string {
  if (!frame.has_timing) return '—'
  return frame.duration_ms != null ? frame.duration_ms.toFixed(3) : '—'
}

function drawFrameMs(frame: { time_us?: number; has_timing?: boolean }): string {
  if (!frame.has_timing) return '—'
  return `${ms(frame.time_us)} ms`
}

function analysisMs(us?: number): string {
  if (!parseResult.value?.has_timing) return '—'
  return ms(us)
}

function pct(value?: number): string {
  if (value == null) return '—'
  const normalized = Math.max(0, Math.min(1, value))
  return `${Math.round(normalized * 100)}%`
}

function numberText(value?: number): string {
  if (value == null) return '—'
  return value.toLocaleString()
}

function shaderKind(shader: { kind?: string }): string {
  return shader.kind === 'api_stat' ? 'API 统计' : '源码'
}

function shaderName(shader: { id?: number; api_name?: string; kind?: string }): string {
  if (shader.kind === 'api_stat') return shader.api_name || '—'
  return shader.id != null ? `Shader #${shader.id}` : 'Shader'
}

function shaderTime(shader: { time_us?: number }): string {
  if (!shader.time_us) return '—'
  return shader.time_us.toLocaleString()
}

function shaderCount(shader: { count?: number }): string {
  if (!shader.count) return '—'
  return shader.count.toLocaleString()
}

function downloadTopFrame(event: MouseEvent, frame: any) {
  event.stopPropagation()
  downloadFrameLog(frame)
}
</script>

<template>
  <div class="analysis-workbench">
    <div v-if="overview" class="analysis-summary">
      <div class="analysis-headline">
        <h3>综合分析</h3>
        <p>{{ overview.summary }}</p>
      </div>
      <div class="mini-stat-grid">
        <div class="mini-stat">
          <span class="mini-label">帧数</span>
          <strong>{{ overview.basic.frame_count }}</strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">P95 帧耗时</span>
          <strong>{{ analysisMs(overview.performance.frame_time.p95_us) }}<small v-if="parseResult?.has_timing">ms</small></strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">P99 帧耗时</span>
          <strong>{{ analysisMs(overview.performance.frame_time.p99_us) }}<small v-if="parseResult?.has_timing">ms</small></strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">API 调用</span>
          <strong>{{ numberText(overview.resources.total_api_calls) }}</strong>
        </div>
      </div>
    </div>

    <div class="analyze-grid">
      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19V5"/><path d="M4 19h16"/><path d="M8 16l3-4 3 2 4-7"/>
            </svg>
            瓶颈判断
          </h4>
        </div>
        <div v-if="bottleneck" class="metric-panel">
          <div class="bottleneck-badge">{{ bottleneckLabel }}</div>
          <div class="metric-row">
            <span>置信度</span>
            <strong>{{ pct(bottleneck.confidence) }}</strong>
          </div>
          <div class="ratio-line">
            <span>Swap 占比</span>
            <div class="ratio-track"><div class="ratio-fill" :style="{ width: pct(bottleneck.swap_ratio) }"></div></div>
            <strong>{{ pct(bottleneck.swap_ratio) }}</strong>
          </div>
          <div class="ratio-line">
            <span>API 占比</span>
            <div class="ratio-track"><div class="ratio-fill secondary" :style="{ width: pct(bottleneck.api_ratio) }"></div></div>
            <strong>{{ pct(bottleneck.api_ratio) }}</strong>
          </div>
          <div class="metric-row">
            <span>稳定性 CV</span>
            <strong>{{ bottleneck.stability.toFixed(3) }}</strong>
          </div>
          <div class="metric-row">
            <span>主要瓶颈</span>
            <strong class="mono">{{ bottleneck.top_bottleneck || '—' }}</strong>
          </div>
          <p v-if="overview" class="hint-text">{{ overview.performance.bottleneck_hint }}</p>
          <p v-if="bottleneck.details" class="hint-text">{{ bottleneck.details }}</p>
        </div>
      </div>

      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 3v18h18"/><rect x="7" y="10" width="3" height="7"/><rect x="12" y="6" width="3" height="11"/><rect x="17" y="13" width="3" height="4"/>
            </svg>
            DrawCall
          </h4>
        </div>
        <div v-if="drawCallSummary" class="metric-panel">
          <div class="metric-row">
            <span>总 DrawCall</span>
            <strong>{{ numberText(drawCallSummary.total_draw_calls) }}</strong>
          </div>
          <div class="metric-row">
            <span>平均每帧</span>
            <strong>{{ drawCallSummary.draw_calls_per_frame_avg.toFixed(1) }}</strong>
          </div>
          <div class="type-chips">
            <span v-for="(count, type) in drawCallSummary.by_type" :key="type">{{ type }}: {{ count }}</span>
          </div>
          <div class="compact-table">
            <div class="compact-row header">
              <span>热帧</span><span>DrawCall</span><span>耗时</span>
            </div>
            <div v-for="frame in hotDrawFrames" :key="frame.frame_num" class="compact-row">
              <span>#{{ frame.frame_num }}</span>
              <span>{{ frame.total_draw_calls }}</span>
              <span>{{ drawFrameMs(frame) }}</span>
            </div>
          </div>
          <p v-if="!drawCallSummary.has_timing" class="hint-text">当前日志没有真实帧耗时，热帧按 DrawCall 数排序。</p>
        </div>
      </div>

      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
            </svg>
            纹理资源
          </h4>
        </div>
        <div v-if="textureSummary" class="metric-panel">
          <div class="metric-row">
            <span>总纹理</span>
            <strong>{{ textureSummary.total_textures }}</strong>
          </div>
          <div class="metric-row">
            <span>活跃纹理</span>
            <strong>{{ textureSummary.active_textures }}</strong>
          </div>
          <div class="metric-row">
            <span>推断纹理</span>
            <strong>{{ textureSummary.inferred_count }}</strong>
          </div>
          <div class="metric-row">
            <span>疑似泄漏</span>
            <strong>{{ textureSummary.leaked_textures.length }}</strong>
          </div>
          <div class="type-chips">
            <span v-for="(count, target) in textureSummary.by_target" :key="target">{{ target }}: {{ count }}</span>
          </div>
          <div class="compact-table">
            <div class="compact-row header">
              <span>Texture</span><span>Target</span><span>Frame</span>
            </div>
            <div v-for="tex in textureSummary.leaked_textures.slice(0, 8)" :key="tex.id" class="compact-row">
              <span>#{{ tex.id }}</span>
              <span>{{ tex.target }}</span>
              <span>{{ tex.frame_num ?? '—' }}</span>
            </div>
          </div>
        </div>
      </div>

    </div>

    <div class="analyze-grid lower-grid">
      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
            </svg>
            Top N 帧
          </h4>
        </div>
        <div class="table-wrap" style="border:none;border-radius:0;">
          <table class="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>帧号 / Frame ID</th>
                <th>耗时 (ms)</th>
                <th>swapBuffers (ms)</th>
                <th>API 耗时 (ms)</th>
                <th>其他耗时 (ms)</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(frame, idx) in topFrames"
                :key="frame.id"
                @click="selectTopFrame(frame)"
                :class="{ selected: selectedTopFrame && selectedTopFrame.id === frame.id }"
              >
                <td>{{ idx + 1 }}</td>
                <td class="frame-id">#{{ frame.id }}</td>
                <td class="mono">{{ frameMs(frame) }}</td>
                <td class="mono">{{ frame.has_timing && frame.swapbuffers_ms != null ? frame.swapbuffers_ms.toFixed(3) : '—' }}</td>
                <td class="mono">{{ frame.has_timing && frame.api_ms != null ? frame.api_ms.toFixed(3) : '—' }}</td>
                <td class="mono">{{ frame.has_timing && frame.other_ms != null ? frame.other_ms.toFixed(3) : '—' }}</td>
                <td>
                  <button class="btn btn-sm btn-default" @click="downloadTopFrame($event, frame)">下载帧日志</button>
                </td>
              </tr>
              <tr v-if="topFrames.length === 0">
                <td colspan="7" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无数据</td>
              </tr>
            </tbody>
          </table>
        </div>

        <p class="hint-text" style="margin:0 1rem 1rem;">点击 Top 帧会打开与帧列表一致的详情弹窗，包含分类统计、关键 API 和原始日志下载。</p>
      </div>

      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="12 2 2 7 12 12 22 7 12 2"/>
              <polyline points="2 17 12 22 22 17"/>
              <polyline points="2 12 12 17 22 12"/>
            </svg>
            Shader / Program 统计
          </h4>
        </div>
        <div class="table-wrap" style="border:none;border-radius:0;">
          <table class="data-table">
            <thead>
              <tr>
                <th>对象 / API</th>
                <th>类型</th>
                <th>调用次数</th>
                <th>总耗时 (μs)</th>
                <th>内容 / Source Preview</th>
                <th>操作 / Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="shader in shaderStats" :key="`${shader.kind || 'source'}-${shader.api_name || shader.id}`">
                <td class="mono">{{ shaderName(shader) }}</td>
                <td>{{ shaderKind(shader) }}</td>
                <td class="mono">{{ shaderCount(shader) }}</td>
                <td class="mono">{{ shaderTime(shader) }}</td>
                <td class="mono">
                  <span v-if="shader.kind === 'api_stat'">{{ shader.source }}</span>
                  <span v-else-if="!shader.expanded">{{ (shader.source || '').substring(0, 50) }}{{ (shader.source || '').length > 50 ? '...' : '' }}</span>
                  <pre v-else class="shader-source">{{ shader.command_line }}&#10;####&#10;{{ shader.source || '' }}&#10;####</pre>
                </td>
                <td>
                  <button v-if="shader.kind !== 'api_stat'" class="btn btn-sm btn-default" @click="toggleShaderExpand(shader)">
                    {{ shader.expanded ? '收起 / Collapse' : '展开 / Expand' }}
                  </button>
                  <span v-else class="hint-text">—</span>
                </td>
              </tr>
              <tr v-if="shaderStats.length === 0">
                <td colspan="6" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无 Shader/Program 数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
