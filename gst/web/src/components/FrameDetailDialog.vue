<script setup lang="ts">
import { computed, inject } from 'vue'
import { Download } from '@element-plus/icons-vue'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const {
  frameModalVisible,
  modalFrame,
  modalFrameDetail,
  modalFrameRawLines,
  modalFrameRawPage,
  modalFrameRawPageSize,
  modalFrameRawTotal,
  modalFrameRawLoading,
  modalFrameRawStartLine,
  modalFrameRawEndLine,
  frameLogDownloading,
} = ctx

const { closeFrameModal, loadModalFrameRawLines, downloadFrameLog } = ctx

const rawText = computed(() => (modalFrameRawLines.value || []).join('\n'))

function ms(us?: number | null): string {
  if (!us) return '—'
  return `${(us / 1000).toFixed(3)} ms`
}

function usText(value?: number | null): string {
  if (!value) return '—'
  return value.toLocaleString()
}

function topApiText(items: any[] = []): string {
  if (!items.length) return '—'
  return items.map(api => `${api.api_name}(${api.count})`).join('  ')
}

function avgUs(row: { call_count?: number; total_time_us?: number; avg_time_us?: number }): string {
  if (row.avg_time_us != null) return usText(row.avg_time_us)
  if (!row.call_count || !row.total_time_us) return '—'
  return usText(Math.floor(row.total_time_us / row.call_count))
}
</script>

<template>
  <el-dialog
    v-model="frameModalVisible"
    :title="modalFrame ? `帧详情 / Frame #${modalFrame.id}` : '帧详情'"
    width="min(1180px, 96vw)"
    class="gst-frame-dialog"
    destroy-on-close
    @closed="closeFrameModal"
  >
    <div v-if="modalFrameDetail" class="frame-dialog-body">
      <div class="frame-dialog-actions">
        <span class="frame-line-range">
          lines {{ modalFrameRawStartLine || modalFrameDetail.start_line }}-{{ modalFrameRawEndLine || modalFrameDetail.end_line }}
        </span>
        <el-button
          type="primary"
          :icon="Download"
          :loading="frameLogDownloading"
          @click="downloadFrameLog(modalFrame)"
        >
          下载完整帧日志
        </el-button>
      </div>

      <div class="frame-kpi-grid">
        <div class="frame-kpi">
          <span>总耗时</span>
          <strong>{{ modalFrameDetail.has_timing ? ms(modalFrameDetail.total_time_us) : '无耗时数据' }}</strong>
        </div>
        <div class="frame-kpi">
          <span>swapBuffers</span>
          <strong>{{ modalFrameDetail.has_timing ? ms(modalFrameDetail.swap_buffer_time_us) : '—' }}</strong>
        </div>
        <div class="frame-kpi">
          <span>API 耗时</span>
          <strong>{{ modalFrameDetail.has_timing ? ms(modalFrameDetail.api_total_time_us) : '—' }}</strong>
        </div>
        <div class="frame-kpi">
          <span>API 调用</span>
          <strong>{{ modalFrameDetail.api_count.toLocaleString() }}</strong>
        </div>
        <div class="frame-kpi">
          <span>DrawCall</span>
          <strong>{{ modalFrameDetail.draw_call_count.toLocaleString() }}</strong>
        </div>
      </div>

      <div class="dialog-section-grid">
        <section class="dialog-section">
          <div class="dialog-section-head">
            <h4>OpenGL 分类统计</h4>
          </div>
          <el-table
            :data="modalFrameDetail.stats?.category_stats || []"
            size="small"
            max-height="260"
            empty-text="暂无分类统计"
          >
            <el-table-column prop="label" label="分类" min-width="130" />
            <el-table-column label="调用" width="100" align="right">
              <template #default="{ row }">{{ row.count.toLocaleString() }}</template>
            </el-table-column>
            <el-table-column label="耗时(μs)" width="120" align="right">
              <template #default="{ row }">{{ usText(row.time_us) }}</template>
            </el-table-column>
            <el-table-column label="Top API" min-width="220">
              <template #default="{ row }">
                <span class="mono subtle">{{ topApiText(row.top_apis) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </section>

        <section class="dialog-section">
          <div class="dialog-section-head">
            <h4>关键 API</h4>
          </div>
          <el-table
            :data="modalFrameDetail.stats?.key_apis || []"
            size="small"
            max-height="260"
            empty-text="暂无关键 API 统计"
          >
            <el-table-column prop="api_name" label="API" min-width="150" />
            <el-table-column prop="label" label="分类" min-width="110" />
            <el-table-column label="调用" width="100" align="right">
              <template #default="{ row }">{{ row.count.toLocaleString() }}</template>
            </el-table-column>
            <el-table-column label="耗时(μs)" width="120" align="right">
              <template #default="{ row }">{{ usText(row.time_us) }}</template>
            </el-table-column>
          </el-table>
        </section>
      </div>

      <section class="dialog-section">
        <div class="dialog-section-head">
          <div>
            <h4>完整 API 调用统计</h4>
            <p>来自该帧 profile/API summary；count 和 time 均为日志真实记录。</p>
          </div>
          <span class="raw-count">{{ (modalFrameDetail.func_stats || []).length }} APIs</span>
        </div>
        <el-table
          :data="modalFrameDetail.func_stats || []"
          size="small"
          max-height="320"
          empty-text="暂无 API 统计"
        >
          <el-table-column prop="name" label="API" min-width="190" />
          <el-table-column label="调用次数" width="110" align="right">
            <template #default="{ row }">{{ row.call_count.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="总耗时(μs)" width="130" align="right">
            <template #default="{ row }">{{ usText(row.total_time_us) }}</template>
          </el-table-column>
          <el-table-column label="平均(μs)" width="120" align="right">
            <template #default="{ row }">{{ avgUs(row) }}</template>
          </el-table-column>
        </el-table>
      </section>

      <section class="dialog-section">
        <div class="dialog-section-head">
          <div>
            <h4>原始 API 日志</h4>
            <p>按帧原始文本分页预览，已去除 apitrace 行号前缀，保留 gc/tid/API/参数。</p>
          </div>
          <span class="raw-count">共 {{ modalFrameRawTotal.toLocaleString() }} 行</span>
        </div>

        <div v-loading="modalFrameRawLoading" class="raw-log-panel">
          <pre v-if="rawText" class="raw-log-text">{{ rawText }}</pre>
          <div v-else class="raw-log-empty">暂无原始日志明细</div>
        </div>

        <div class="dialog-pager">
          <el-pagination
            small
            background
            layout="prev, pager, next, jumper, total"
            :current-page="modalFrameRawPage"
            :page-size="modalFrameRawPageSize"
            :total="modalFrameRawTotal"
            :disabled="modalFrameRawLoading"
            @current-change="loadModalFrameRawLines"
          />
        </div>
      </section>
    </div>

    <div v-else class="frame-dialog-loading">
      <el-skeleton :rows="6" animated />
    </div>
  </el-dialog>
</template>
