<script setup lang="ts">
import { computed, inject } from 'vue'
import type { DrawCallInsight, FrameData, ProgramInfo, ShaderObjectInfo } from '../types'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const {
  frames,
  tracePrograms,
  traceLoading,
  traceProgramSummary,
  selectedTraceFrame,
  traceFrameJump,
  frameProgramInsight,
  traceDrawCalls,
  traceDrawCallPage,
  traceDrawCallTotal,
  traceDrawCallTotalPages,
  selectedTraceProgramId,
  selectedTraceProgram,
  programDetailLoading,
} = ctx

const {
  selectTraceFrame,
  selectTraceFrameById,
  loadTraceDrawCalls,
  filterTraceProgram,
  loadTraceProgramDetail,
} = ctx

const topPrograms = computed(() => {
  return [...tracePrograms.value]
    .sort((a: ProgramInfo, b: ProgramInfo) => b.draw_call_count - a.draw_call_count)
    .slice(0, 12)
})

function sourceLabel(type: string): string {
  if (type === 'source') return '源码'
  if (type === 'program_binary') return 'Binary'
  return '未知'
}

function confidenceLabel(confidence: string): string {
  if (confidence === 'high') return 'High'
  if (confidence === 'medium') return 'Medium'
  return 'Low'
}

function confidenceClass(confidence: string): string {
  return `confidence-${confidence || 'low'}`
}

function lineRange(item: { first_line?: number; last_line?: number }): string {
  if (!item.first_line && !item.last_line) return '—'
  if (item.first_line === item.last_line) return String(item.first_line)
  return `${item.first_line ?? '—'}-${item.last_line ?? '—'}`
}

function shaderLabel(shader: ShaderObjectInfo): string {
  return `#${shader.id} ${shader.type}`
}

function shaderIdsText(ids: number[]): string {
  return ids.length ? ids.map((id: number) => `#${id}`).join(', ') : '—'
}

function shaderStack(program: { shaders?: ShaderObjectInfo[]; shader_ids?: number[] }): string {
  const shaders = program.shaders || []
  if (shaders.length) {
    return shaders.map((shader: ShaderObjectInfo) => `#${shader.id} ${shader.type}`).join(' / ')
  }
  return shaderIdsText(program.shader_ids || [])
}

function frameDuration(frame: FrameData): string {
  if (!frame.has_timing) return '—'
  return frame.duration_ms != null ? frame.duration_ms.toFixed(3) : '—'
}

function frameTimingText(): string {
  if (!selectedTraceFrame.value) return '—'
  if (!selectedTraceFrame.value.has_timing) return '无耗时数据'
  return `${frameDuration(selectedTraceFrame.value)} ms`
}

function programDrawShare(draws: number): string {
  const total = frameProgramInsight.value?.total_draw_calls || 0
  if (!total) return '0%'
  return `${Math.round((draws / total) * 100)}%`
}

function textureText(draw: DrawCallInsight): string {
  const entries = Object.entries(draw.textures || {}).filter(([, id]) => id !== 0)
  if (!entries.length) return '—'
  return entries.slice(0, 3).map(([unit, id]) => `${unit}:${id}`).join(', ')
}
</script>

<template>
  <div class="trace-workbench">
    <div class="trace-toolbar">
      <div>
        <h3>Trace Inspector</h3>
        <p>按帧重建 Program、Shader、DrawCall 和关键 GL 状态；耗时只展示日志中真实存在的数据。</p>
      </div>
      <div class="trace-kpis">
        <span><strong>{{ traceProgramSummary.total }}</strong> Programs</span>
        <span><strong>{{ traceProgramSummary.sourcePrograms }}</strong> Source</span>
        <span><strong>{{ traceProgramSummary.binaryPrograms }}</strong> Binary</span>
        <span><strong>{{ traceProgramSummary.totalDraws.toLocaleString() }}</strong> Draws</span>
      </div>
    </div>

    <div v-if="traceLoading" class="trace-loading">加载 Trace Inspector 数据...</div>

    <div class="trace-layout">
      <aside class="trace-rail">
        <div class="trace-panel-title">帧 / Frames</div>
        <div class="trace-jump">
          <input v-model.number="traceFrameJump" class="input" type="number" min="0" placeholder="Frame #">
          <button class="btn btn-sm btn-default" @click="selectTraceFrameById(traceFrameJump)">跳转</button>
        </div>
        <div class="trace-frame-list">
          <button
            v-for="frame in frames"
            :key="frame.id"
            :class="['trace-frame-row', { active: selectedTraceFrame && selectedTraceFrame.id === frame.id }]"
            @click="selectTraceFrame(frame)"
          >
            <span>#{{ frame.id }}</span>
            <small>{{ frame.draw_call_count.toLocaleString() }} draws · {{ frameDuration(frame) }} ms</small>
          </button>
        </div>
      </aside>

      <main class="trace-main">
        <section class="trace-card">
          <div class="trace-card-head">
            <div>
              <h4>Frame Program Summary</h4>
              <p v-if="frameProgramInsight">
                Frame #{{ frameProgramInsight.frame_num }} · {{ frameTimingText() }} · {{ frameProgramInsight.api_call_count.toLocaleString() }} API · {{ frameProgramInsight.total_draw_calls.toLocaleString() }} draw calls · lines {{ frameProgramInsight.start_line }}-{{ frameProgramInsight.end_line }}
              </p>
              <p v-else>选择一帧查看 Program 使用归纳。</p>
            </div>
            <button v-if="selectedTraceProgramId !== null" class="btn btn-sm btn-default" @click="filterTraceProgram(null)">清除 Program 过滤</button>
          </div>

          <div class="table-wrap trace-table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Program</th>
                  <th>Draw Calls</th>
                  <th>占比</th>
                  <th>Use</th>
                  <th>来源</th>
                  <th>本帧 Shader</th>
                  <th>置信度</th>
                  <th>Line</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="program in frameProgramInsight?.programs || []"
                  :key="program.program_id"
                  :class="{ selected: selectedTraceProgramId === program.program_id }"
                  @click="filterTraceProgram(program.program_id)"
                >
                  <td class="mono">#{{ program.program_id }}</td>
                  <td class="mono">{{ program.draw_call_count }}</td>
                  <td class="mono">{{ programDrawShare(program.draw_call_count) }}</td>
                  <td class="mono">{{ program.use_count }}</td>
                  <td>{{ sourceLabel(program.source_type) }}</td>
                  <td class="mono shader-cell">{{ shaderStack(program) }}</td>
                  <td><span :class="['confidence-pill', confidenceClass(program.confidence)]">{{ confidenceLabel(program.confidence) }}</span></td>
                  <td class="mono">{{ lineRange(program) }}</td>
                </tr>
                <tr v-if="!frameProgramInsight || frameProgramInsight.programs.length === 0">
                  <td colspan="8" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无 Program 使用数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="trace-card">
          <div class="trace-card-head">
            <div>
              <h4>DrawCall Timeline</h4>
              <p>按当前帧和 Program 过滤展示 draw call 归属与关键状态。</p>
            </div>
            <div class="trace-pager">
              <button class="page-btn" @click="loadTraceDrawCalls(traceDrawCallPage - 1)" :disabled="traceDrawCallPage <= 1">‹</button>
              <span>{{ traceDrawCallPage }} / {{ traceDrawCallTotalPages }} · {{ traceDrawCallTotal }} 条</span>
              <button class="page-btn" @click="loadTraceDrawCalls(traceDrawCallPage + 1)" :disabled="traceDrawCallPage >= traceDrawCallTotalPages">›</button>
            </div>
          </div>

          <div class="table-wrap trace-table-wrap draw-table">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Line</th>
                  <th>#</th>
                  <th>API</th>
                  <th>Program</th>
                  <th>VAO</th>
                  <th>VBO/EBO</th>
                  <th>FBO</th>
                  <th>Texture</th>
                  <th>Params</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="draw in traceDrawCalls" :key="`${draw.line_num}-${draw.api_name}`">
                  <td class="mono">{{ draw.line_num }}</td>
                  <td class="mono">{{ draw.index }}</td>
                  <td class="mono">{{ draw.api_name }}</td>
                  <td class="mono">#{{ draw.program_id || '—' }}</td>
                  <td class="mono">{{ draw.vao || '—' }}</td>
                  <td class="mono">{{ draw.array_buffer || '—' }}/{{ draw.element_buffer || '—' }}</td>
                  <td class="mono">{{ draw.fbo || '—' }}</td>
                  <td class="mono">{{ textureText(draw) }}</td>
                  <td class="mono params-cell">{{ draw.raw_params }}</td>
                </tr>
                <tr v-if="traceDrawCalls.length === 0">
                  <td colspan="9" style="text-align:center;color:var(--text-placeholder);padding:1.5rem;">暂无 DrawCall 数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </main>

      <aside class="trace-inspector">
        <section class="trace-card">
          <div class="trace-card-head compact">
            <h4>Program Detail</h4>
          </div>
          <div v-if="programDetailLoading" class="trace-loading small">加载 Program...</div>
          <div v-else-if="selectedTraceProgram" class="program-detail">
            <div class="program-title">
              <strong>Program #{{ selectedTraceProgram.id }}</strong>
              <span :class="['confidence-pill', confidenceClass(selectedTraceProgram.confidence)]">{{ confidenceLabel(selectedTraceProgram.confidence) }}</span>
            </div>
            <div class="detail-kv"><span>来源</span><strong>{{ sourceLabel(selectedTraceProgram.source_type) }}</strong></div>
            <div class="detail-kv"><span>Draw Calls</span><strong>{{ selectedTraceProgram.draw_call_count.toLocaleString() }}</strong></div>
            <div class="detail-kv"><span>Frames</span><strong>{{ selectedTraceProgram.frames_used.length }}</strong></div>
            <div class="detail-kv"><span>Line</span><strong>{{ lineRange(selectedTraceProgram) }}</strong></div>
            <div v-if="selectedTraceProgram.source_type === 'program_binary'" class="binary-box">
              <span>Binary Format</span>
              <strong>{{ selectedTraceProgram.binary_format || '—' }}</strong>
              <span>Binary Size</span>
              <strong>{{ selectedTraceProgram.binary_size_bytes || 0 }} bytes</strong>
            </div>
            <div v-if="selectedTraceProgram.shaders?.length" class="shader-stack">
              <div v-for="shader in selectedTraceProgram.shaders" :key="shader.id" class="shader-box">
                <div class="shader-head">
                  <strong>{{ shaderLabel(shader) }}</strong>
                  <span>{{ shader.source_available ? 'source available' : 'no source in log' }}</span>
                </div>
                <pre v-if="shader.source_available" class="shader-source">{{ shader.source }}</pre>
              </div>
            </div>
          </div>
          <div v-else class="empty-state trace-empty">
            <p>选择 Program 查看源码或 Binary 溯源</p>
          </div>
        </section>

        <section class="trace-card">
          <div class="trace-card-head compact">
            <h4>Top Programs</h4>
          </div>
          <button
            v-for="program in topPrograms"
            :key="program.id"
            :class="['program-rank-row', { active: selectedTraceProgramId === program.id }]"
            @click="loadTraceProgramDetail(program.id)"
          >
            <span class="mono">#{{ program.id }}</span>
            <small>{{ program.draw_call_count.toLocaleString() }} draws · {{ sourceLabel(program.source_type) }}</small>
          </button>
        </section>
      </aside>
    </div>
  </div>
</template>
