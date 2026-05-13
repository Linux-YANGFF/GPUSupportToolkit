<script setup lang="ts">
import { computed, inject } from 'vue'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const {
  overview,
  bottleneck,
  drawCallSummary,
  textureSummary,
  hotDrawFrames,
  topFrames,
  shaderStats,
  selectedTopFrame,
  selectedTopFrameFuncStats,
  workflowResults,
  workflowLoading,
  workflowLabels,
} = ctx

const { selectTopFrame, toggleShaderExpand, runWorkflow } = ctx

const workflowIds = ['performance', 'crash', 'rendering', 'memory']

const bottleneckLabel = computed(() => {
  const type = bottleneck.value?.type
  if (type === 'cpu_bound') return 'CPU Bound'
  if (type === 'gpu_bound') return 'GPU Bound'
  if (type === 'balanced') return 'Balanced'
  if (type === 'unstable') return 'Unstable'
  return '—'
})

function ms(us?: number): string {
  if (us == null) return '—'
  return (us / 1000).toFixed(3)
}

function pct(value?: number): string {
  if (value == null) return '—'
  return `${Math.round(value * 100)}%`
}

function numberText(value?: number): string {
  if (value == null) return '—'
  return value.toLocaleString()
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
          <strong>{{ ms(overview.performance.frame_time.p95_us) }}<small>ms</small></strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">P99 帧耗时</span>
          <strong>{{ ms(overview.performance.frame_time.p99_us) }}<small>ms</small></strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">API 调用</span>
          <strong>{{ numberText(overview.resources.total_api_calls) }}</strong>
        </div>
        <div class="mini-stat">
          <span class="mini-label">诊断问题</span>
          <strong>{{ overview.diagnosis_summary.total_findings }}</strong>
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
              <span>{{ ms(frame.time_us) }} ms</span>
            </div>
          </div>
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

      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
            </svg>
            工作流分析
          </h4>
        </div>
        <div class="workflow-panel">
          <div class="workflow-buttons">
            <button
              v-for="id in workflowIds"
              :key="id"
              class="btn btn-sm btn-default"
              :disabled="workflowLoading !== null"
              @click="runWorkflow(id)"
            >
              {{ workflowLoading === id ? '分析中...' : workflowLabels[id] }}
            </button>
          </div>
          <div v-for="id in workflowIds" :key="id" v-show="workflowResults[id]" class="workflow-result">
            <h5>{{ workflowLabels[id] }}: {{ workflowResults[id]?.conclusion }}</h5>
            <ul>
              <li v-for="item in workflowResults[id]?.evidence || []" :key="item">{{ item }}</li>
            </ul>
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
                <td class="mono">{{ frame.duration_ms.toFixed(3) }}</td>
                <td class="mono">{{ frame.swapbuffers_ms != null ? frame.swapbuffers_ms.toFixed(3) : '—' }}</td>
                <td class="mono">{{ frame.api_ms != null ? frame.api_ms.toFixed(3) : '—' }}</td>
                <td class="mono">{{ frame.other_ms != null ? frame.other_ms.toFixed(3) : '—' }}</td>
              </tr>
              <tr v-if="topFrames.length === 0">
                <td colspan="6" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无数据</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="selectedTopFrame" class="detail-panel" style="margin:1rem;border-radius:var(--radius-md);">
          <div class="detail-header">
            <h3>帧 #{{ selectedTopFrame.id }} 函数统计 / Function Stats</h3>
            <button class="btn btn-sm btn-default" @click="selectedTopFrame = null">关闭</button>
          </div>
          <div class="table-wrap" style="border:none;border-radius:0;">
            <table class="data-table">
              <thead>
                <tr>
                  <th>函数名 / Function</th>
                  <th>调用次数 / Count</th>
                  <th>总耗时 (μs)</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="func in selectedTopFrameFuncStats" :key="func.name">
                  <td class="mono">{{ func.name }}</td>
                  <td class="mono">{{ func.call_count }}</td>
                  <td class="mono">{{ func.total_time_us }}</td>
                </tr>
                <tr v-if="!selectedTopFrameFuncStats || selectedTopFrameFuncStats.length === 0">
                  <td colspan="3" style="text-align:center;color:var(--text-placeholder);padding:1rem;">暂无数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="analyze-block">
        <div class="analyze-block-header">
          <h4>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="12 2 2 7 12 12 22 7 12 2"/>
              <polyline points="2 17 12 22 22 17"/>
              <polyline points="2 12 12 17 22 12"/>
            </svg>
            Shader 统计
          </h4>
        </div>
        <div class="table-wrap" style="border:none;border-radius:0;">
          <table class="data-table">
            <thead>
              <tr>
                <th>Shader ID</th>
                <th>源码预览 / Source Preview</th>
                <th>操作 / Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="shader in shaderStats" :key="shader.id">
                <td class="mono">{{ shader.id }}</td>
                <td class="mono">
                  <span v-if="!shader.expanded">{{ (shader.source || '').substring(0, 50) }}{{ (shader.source || '').length > 50 ? '...' : '' }}</span>
                  <pre v-else class="shader-source">{{ shader.command_line }}&#10;####&#10;{{ shader.source || '' }}&#10;####</pre>
                </td>
                <td>
                  <button class="btn btn-sm btn-default" @click="toggleShaderExpand(shader)">
                    {{ shader.expanded ? '收起 / Collapse' : '展开 / Expand' }}
                  </button>
                </td>
              </tr>
              <tr v-if="shaderStats.length === 0">
                <td colspan="3" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
