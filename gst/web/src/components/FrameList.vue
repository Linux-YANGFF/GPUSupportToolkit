<script setup lang="ts">
import { inject } from 'vue'
import type { FrameData } from '../types'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const { frames, currentPage, pageSize, totalFrames, totalPages, jumpPage, pageRange } = ctx
const { frameModalVisible, modalFrame, modalFrameDetail } = ctx
const { openFrameModal, closeFrameModal, goPage } = ctx

function formatDuration(frame: FrameData): string {
  if (frame.duration_ms != null) return frame.duration_ms.toFixed(3)
  if (frame.duration_us != null) return (frame.duration_us / 1000).toFixed(3)
  return '—'
}

function formatMs(value: number | null | undefined): string {
  if (value != null) return value.toFixed(3)
  return '—'
}
</script>

<template>
  <div>
    <div class="panel-header">
      <h3>帧列表</h3>
      <span class="count-tag">{{ totalFrames }} 帧 / 共 {{ totalPages }} 页</span>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>帧号 / Frame ID</th>
            <th>耗时 (ms) / Duration</th>
            <th>swapBuffers (ms)</th>
            <th>API 耗时 (ms) / API Time</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="frame in frames"
            :key="frame.id"
            @click="openFrameModal(frame)"
          >
            <td class="frame-id">#{{ frame.id }}</td>
            <td class="mono">{{ formatDuration(frame) }}</td>
            <td class="mono">{{ frame.swapbuffers_ms != null ? frame.swapbuffers_ms.toFixed(3) : '—' }}</td>
            <td class="mono">{{ frame.api_ms != null ? frame.api_ms.toFixed(3) : '—' }}</td>
          </tr>
          <tr v-if="frames.length === 0">
            <td colspan="4" style="text-align:center;color:var(--text-placeholder);padding:2rem;">暂无数据</td>
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

    <div v-if="frameModalVisible && modalFrame" class="modal-overlay" @click.self="closeFrameModal">
      <div class="modal-dialog">
        <div class="modal-header">
          <h3>帧详情 / Frame Detail #{{ modalFrame.id }}</h3>
          <button class="btn btn-sm btn-default" @click="closeFrameModal">关闭</button>
        </div>
        <div v-if="modalFrameDetail" class="modal-body">
          <div class="detail-grid">
            <div class="detail-item">
              <div class="label">帧号 / Frame ID</div>
              <div class="value">#{{ modalFrameDetail.frame_num }}</div>
            </div>
            <div class="detail-item">
              <div class="label">总耗时 / Total Duration</div>
              <div class="value">{{ (modalFrameDetail.total_time_us / 1000).toFixed(3) }} ms</div>
            </div>
            <div class="detail-item">
              <div class="label">swapBuffers</div>
              <div class="value">{{ modalFrameDetail.swap_buffer_time_us ? (modalFrameDetail.swap_buffer_time_us / 1000).toFixed(3) : '—' }} ms</div>
            </div>
            <div class="detail-item">
              <div class="label">API 总耗时 / API Time</div>
              <div class="value">{{ modalFrameDetail.api_total_time_us ? (modalFrameDetail.api_total_time_us / 1000).toFixed(3) : '—' }} ms</div>
            </div>
          </div>
          <div v-if="modalFrameDetail.func_stats.length" class="api-list-header">API 调用统计 / API Summary</div>
          <div v-if="modalFrameDetail.func_stats.length" class="table-wrap" style="max-height:320px;overflow-y:auto;">
            <table class="data-table">
              <thead>
                <tr>
                  <th>函数名 / Function</th>
                  <th>调用次数 / Count</th>
                  <th>耗时 (μs) / Time</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="call in modalFrameDetail.func_stats" :key="call.name">
                  <td class="mono">{{ call.name }}</td>
                  <td class="mono">{{ call.call_count }}</td>
                  <td class="mono">{{ call.total_time_us }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="modal-body">
          <div class="empty-state" style="padding:2rem;">
            <p>加载帧详情...</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
