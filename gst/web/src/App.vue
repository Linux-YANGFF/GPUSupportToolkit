<script setup lang="ts">
import { provide } from 'vue'
import { useLogAnalysis } from './composables/useLogAnalysis'
import FileUpload from './components/FileUpload.vue'
import AnalysisTabs from './components/AnalysisTabs.vue'
import FrameList from './components/FrameList.vue'
import FunctionStats from './components/FunctionStats.vue'
import SearchPanel from './components/SearchPanel.vue'
import TraceInspector from './components/TraceInspector.vue'
import FrameDetailDialog from './components/FrameDetailDialog.vue'

const ctx = useLogAnalysis()
provide('ctx', ctx)

const {
  filePath, selectedFile, parsing, parseResult,
  loading, loadingText, toast, stopping,
  browseFile, parseFile, resetAll, stopService,
  exportFormat, exporting, exportType, exportData,
  activeTab, tabs, switchTab,
} = ctx
</script>

<template>
  <div class="page-wrapper">
    <div class="page-title">
      <h2>日志分析</h2>
      <p>上传完整 apitrace 日志或输入路径，解析后查看每帧 OpenGL 统计、Shader/Program、搜索与导出。</p>
    </div>

    <FileUpload
      :filePath="filePath"
      :parsing="parsing"
      :stopped="stopping"
      @update:filePath="filePath = $event"
      @browse="browseFile"
      @parse="parseFile"
      @reset="resetAll"
      @stop="stopService"
    />
    <div v-if="selectedFile" class="card" style="margin-top:-0.5rem;margin-bottom:1.25rem;font-size:0.8125rem;color:var(--text-secondary);">
      已选择文件: <strong>{{ selectedFile.name }}</strong>
    </div>

    <div v-if="parseResult" class="stats-row">
      <div class="stat-card">
        <div class="stat-label">日志格式</div>
        <div class="stat-value">{{ parseResult.format || '—' }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">帧数</div>
        <div class="stat-value">{{ parseResult.frame_count ?? 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">FPS</div>
        <div class="stat-value">{{ parseResult.fps ?? 0 }}<small>fps</small></div>
      </div>
      <div class="stat-card">
        <div class="stat-label">最大帧耗时</div>
        <div class="stat-value">
          {{ parseResult.has_timing ? (parseResult.max_frame_time ?? 0) : '—' }}<small v-if="parseResult.has_timing">ms</small>
        </div>
      </div>
    </div>

    <AnalysisTabs v-if="parseResult">
      <template #frames>
        <FrameList />
      </template>
      <template #search>
        <SearchPanel />
      </template>
      <template #analyze>
        <FunctionStats />
      </template>
      <template #trace>
        <TraceInspector />
      </template>
      <template #export>
        <div class="export-section">
          <h3>导出日志数据 / Export Log Data</h3>

          <div class="export-options" style="margin-bottom:1.25rem;">
            <div style="font-size:0.8125rem;font-weight:600;color:var(--text-secondary);margin-bottom:0.625rem;text-transform:uppercase;letter-spacing:0.04em;">导出类型 / Export Type</div>
            <div style="display:flex;gap:1rem;flex-wrap:wrap;">
              <label class="radio-option">
                <input type="radio" v-model="exportType" value="frames">
                <span>所有帧 / Frames</span>
              </label>
              <label class="radio-option">
                <input type="radio" v-model="exportType" value="top">
                <span>Top帧 / Top Frames</span>
              </label>
              <label class="radio-option">
                <input type="radio" v-model="exportType" value="shader">
                <span>Shader</span>
              </label>
              <label class="radio-option">
                <input type="radio" v-model="exportType" value="longest">
                <span>最长帧 / Longest</span>
              </label>
            </div>
          </div>

          <div style="font-size:0.8125rem;font-weight:600;color:var(--text-secondary);margin-bottom:0.625rem;text-transform:uppercase;letter-spacing:0.04em;">导出格式 / Export Format</div>
          <div class="radio-group">
            <label class="radio-option">
              <input type="radio" v-model="exportFormat" value="json">
              <span>JSON</span>
            </label>
            <label class="radio-option">
              <input type="radio" v-model="exportFormat" value="csv">
              <span>CSV</span>
            </label>
            <label class="radio-option">
              <input type="radio" v-model="exportFormat" value="txt">
              <span>TXT</span>
            </label>
          </div>
          <button class="btn btn-primary btn-lg" @click="exportData" :disabled="!parseResult || exporting">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            导出 / Export {{ exportFormat.toUpperCase() }}
          </button>
        </div>
      </template>
    </AnalysisTabs>

    <FrameDetailDialog />

    <div v-if="!parseResult" class="empty-state" style="margin-top:2rem;">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
      </svg>
      <p>还没有解析日志</p>
      <small>输入文件路径或上传文件，点击解析开始</small>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>{{ loadingText }}</p>
    </div>

    <div v-if="toast" :class="['toast', toast.type]">
      <div class="toast-icon">
        <svg v-if="toast.type === 'error'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
          <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
      </div>
      <div class="toast-body">
        <p>{{ toast.message }}</p>
      </div>
      <button class="btn btn-sm" style="border:none;background:none;padding:0;" @click="toast = null">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>
  </div>
</template>
