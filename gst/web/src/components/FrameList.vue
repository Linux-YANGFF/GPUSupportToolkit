<script setup lang="ts">
import { inject } from 'vue'
import type { FrameData } from '../types'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const { frames, currentPage, pageSize, totalFrames, totalPages, jumpPage, pageRange } = ctx
const { openFrameModal, goPage, downloadFrameLog } = ctx

function formatDuration(frame: FrameData): string {
  if (!frame.has_timing) return '—'
  if (frame.duration_ms != null) return frame.duration_ms.toFixed(3)
  if (frame.duration_us != null) return (frame.duration_us / 1000).toFixed(3)
  return '—'
}

function formatMs(value: number | null | undefined, hasTiming = true): string {
  if (!hasTiming) return '—'
  if (value != null) return value.toFixed(3)
  return '—'
}

function keyApiCount(frame: FrameData, key: string): string {
  const item = frame.key_apis?.find(api => api.key === key || api.api_name === key)
  return item ? item.count.toLocaleString() : '0'
}

function keyApiTime(frame: FrameData, key: string): string {
  const item = frame.key_apis?.find(api => api.key === key || api.api_name === key)
  if (!item || !item.time_us) return '—'
  return (item.time_us / 1000).toFixed(3)
}

function categorySummary(frame: FrameData, category: string): string {
  const item = frame.category_stats?.find(stat => stat.category === category)
  if (!item) return '0'
  const time = item.time_us ? ` / ${(item.time_us / 1000).toFixed(2)}ms` : ''
  return `${item.count.toLocaleString()}${time}`
}

function openDownload(event: MouseEvent, frame: FrameData) {
  event.stopPropagation()
  downloadFrameLog(frame)
}
</script>

<template>
  <div class="frame-list-workbench">
    <div class="panel-header">
      <div>
        <h3>帧列表</h3>
        <p class="section-subtitle">逐帧查看真实耗时、DrawCall、关键 API 和原始日志。</p>
      </div>
      <span class="count-tag">{{ totalFrames }} 帧 / 共 {{ totalPages }} 页</span>
    </div>

    <div class="table-wrap">
      <table class="data-table frame-table">
        <thead>
          <tr>
            <th>帧号</th>
            <th>耗时(ms)</th>
            <th>swap(ms)</th>
            <th>API(ms)</th>
            <th>API 调用</th>
            <th>DrawCall</th>
            <th>glDrawElements</th>
            <th>glDrawArrays</th>
            <th>数据传输</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="frame in frames"
            :key="frame.id"
            class="clickable-row"
            @click="openFrameModal(frame)"
          >
            <td class="frame-id">#{{ frame.id }}</td>
            <td class="mono strong">{{ formatDuration(frame) }}</td>
            <td class="mono">{{ formatMs(frame.swapbuffers_ms, frame.has_timing) }}</td>
            <td class="mono">{{ formatMs(frame.api_ms, frame.has_timing) }}</td>
            <td class="mono">{{ frame.api_count.toLocaleString() }}</td>
            <td class="mono">{{ frame.draw_call_count.toLocaleString() }}</td>
            <td class="mono">
              {{ keyApiCount(frame, 'glDrawElements') }}
              <small v-if="keyApiTime(frame, 'glDrawElements') !== '—'"> / {{ keyApiTime(frame, 'glDrawElements') }}ms</small>
            </td>
            <td class="mono">
              {{ keyApiCount(frame, 'glDrawArrays') }}
              <small v-if="keyApiTime(frame, 'glDrawArrays') !== '—'"> / {{ keyApiTime(frame, 'glDrawArrays') }}ms</small>
            </td>
            <td class="mono transfer-cell">
              <span>Buffer {{ categorySummary(frame, 'buffer') }}</span>
              <span>Texture {{ categorySummary(frame, 'texture') }}</span>
            </td>
            <td>
              <button class="btn btn-sm btn-default" @click="openDownload($event, frame)">下载帧日志</button>
            </td>
          </tr>
          <tr v-if="frames.length === 0">
            <td colspan="10" style="text-align:center;color:var(--text-placeholder);padding:2rem;">暂无数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination">
      <div class="pagination-info">
        第 {{ currentPage }} / {{ totalPages }} 页，每页 {{ pageSize }} 条，共 {{ totalFrames }} 条
      </div>
      <div class="pagination-controls">
        <button class="page-btn" @click="goPage(1)" :disabled="currentPage <= 1" title="首页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="11 17 6 12 11 7"/><polyline points="18 17 13 12 18 7"/>
          </svg>
        </button>
        <button class="page-btn" @click="goPage(currentPage - 1)" :disabled="currentPage <= 1" title="上一页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>
        <template v-for="p in pageRange" :key="p">
          <button v-if="p !== '...'" class="page-btn" :class="{ active: p === currentPage }" @click="goPage(p)">{{ p }}</button>
          <span v-else style="padding:0 0.25rem;color:var(--text-placeholder);">...</span>
        </template>
        <button class="page-btn" @click="goPage(currentPage + 1)" :disabled="currentPage >= totalPages" title="下一页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
        <button class="page-btn" @click="goPage(totalPages)" :disabled="currentPage >= totalPages" title="末页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="13 17 18 12 13 7"/><polyline points="6 17 11 12 6 7"/>
          </svg>
        </button>
        <div class="page-jump">
          <span>跳至</span>
          <input type="number" v-model.number="jumpPage" class="input" min="1" :max="totalPages" @keyup.enter="goPage(jumpPage)">
          <span>页</span>
          <button class="btn btn-sm btn-default" @click="goPage(jumpPage)">跳转</button>
        </div>
      </div>
    </div>
  </div>
</template>
