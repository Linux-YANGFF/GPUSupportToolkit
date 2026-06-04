<script setup lang="ts">
import { ref, computed } from 'vue'

interface Finding {
  id?: string
  severity: string
  severity_rank?: number
  category?: string
  category_label?: string
  kind?: string
  confidence?: string
  count?: number
  examples?: string[]
  description: string
  evidence: string
  root_cause_chain: string[]
  fix_suggestion: string
}

interface DiagnosisSummary {
  total_findings: number
  critical_count: number
  high_count: number
  medium_count: number
  low_count: number
  info_count: number
}

interface DiagnosisReport {
  schema_version?: string
  source_file: string
  generated_at: string
  summary: DiagnosisSummary
  findings: Finding[]
}

const props = defineProps<{
  filePath: string
}>()

const TOAST_TIMEOUT = 5000

const diagnosing = ref(false)
const progressText = ref('')
const report = ref<DiagnosisReport | null>(null)
const toast = ref<{ message: string; type: 'error' | 'success' } | null>(null)
const selectedSeverity = ref('key')
const showAdvisory = ref(false)
const activeFinding = ref<number | null>(null)

const severityOrder = ['critical', 'high', 'medium', 'low', 'info'] as const
const keySeverities = new Set(['critical', 'high', 'medium'])
const severityLabels: Record<string, string> = {
  critical: '严重 (Critical)',
  high: '高 (High)',
  medium: '中 (Medium)',
  low: '低 (Low)',
  info: '信息 (Info)',
}
const severityColors: Record<string, string> = {
  critical: 'var(--danger)',
  high: '#fa8c16',
  medium: 'var(--warning)',
  low: 'var(--primary)',
  info: 'var(--text-secondary)',
}
const severityBgs: Record<string, string> = {
  critical: 'var(--danger-light)',
  high: '#fff7e6',
  medium: '#fffbe6',
  low: 'var(--primary-light)',
  info: 'var(--bg-page)',
}

const visibleFindings = computed(() => {
  if (!report.value) return []
  if (selectedSeverity.value === 'key') {
    return rankedFindings(report.value.findings).filter(f => keySeverities.has(f.severity))
  }
  if (selectedSeverity.value === 'all') {
    const ranked = rankedFindings(report.value.findings)
    return showAdvisory.value ? ranked : ranked.filter(f => keySeverities.has(f.severity))
  }
  return report.value.findings.filter(f => f.severity === selectedSeverity.value)
})

const keyFindingCount = computed(() => {
  if (!report.value) return 0
  return report.value.findings.filter(f => keySeverities.has(f.severity)).length
})

const advisoryFindingCount = computed(() => {
  if (!report.value) return 0
  return report.value.summary.low_count + report.value.summary.info_count
})

function rankedFindings(findings: Finding[]): Finding[] {
  const ranked: Finding[] = []
  for (const sev of severityOrder) {
    for (const f of findings) {
      if (f.severity === sev) ranked.push(f)
    }
  }
  return ranked
}

const markdownReport = computed(() => {
  if (!report.value) return ''
  const r = report.value
  let md = '# GPU 诊断报告\n\n'
  md += `**源文件**: ${r.source_file}\n`
  md += `**生成时间**: ${r.generated_at}\n`
  md += `**发现问题总数**: ${r.summary.total_findings}\n\n`
  md += '---\n\n## 摘要\n\n'

  const summaryItems = [
    { label: '严重 (Critical)', count: r.summary.critical_count },
    { label: '高 (High)', count: r.summary.high_count },
    { label: '中 (Medium)', count: r.summary.medium_count },
    { label: '低 (Low)', count: r.summary.low_count },
    { label: '信息 (Info)', count: r.summary.info_count },
  ]
  for (const s of summaryItems) {
    md += `- **${s.label}**: ${s.count}\n`
  }
  md += '\n---\n\n'

  const severitySections = [
    { label: '严重问题 (Critical)', sev: 'critical' },
    { label: '高风险问题 (High)', sev: 'high' },
    { label: '中风险问题 (Medium)', sev: 'medium' },
    { label: '低风险问题 (Low)', sev: 'low' },
    { label: '信息 (Info)', sev: 'info' },
  ]

  for (const sec of severitySections) {
    const list = r.findings.filter(f => f.severity === sec.sev)
    if (list.length === 0) continue
    md += `## ${sec.label}\n\n`
    list.forEach((f, i) => {
      md += `### ${i + 1}. ${f.description}\n\n`
      if (f.category || f.category_label) md += `- **类别**: ${f.category_label || f.category}\n`
      if (f.kind) md += `- **类型**: ${f.kind}\n`
      if (f.confidence) md += `- **置信度**: ${f.confidence}\n`
      if (f.count) md += `- **数量**: ${f.count}\n`
      md += `- **严重程度**: ${f.severity}\n`
      if (f.evidence) md += `- **证据**: ${f.evidence}\n`
      if (f.examples && f.examples.length > 0) {
        md += '- **示例**:\n'
        f.examples.forEach(example => {
          md += `  - ${example}\n`
        })
      }
      if (f.root_cause_chain && f.root_cause_chain.length > 0) {
        md += '- **根因链**:\n'
        f.root_cause_chain.forEach(rc => {
          md += `  1. ${rc}\n`
        })
      }
      if (f.fix_suggestion) md += `- **修复建议**: ${f.fix_suggestion}\n`
      md += '\n'
    })
    md += '---\n\n'
  }
  return md
})

function showToast(message: string, type: 'error' | 'success' = 'error') {
  toast.value = { message, type }
  setTimeout(() => { toast.value = null }, TOAST_TIMEOUT)
}

const progressSteps = [
  '初始化诊断引擎...',
  '执行空指针排查分析...',
  '执行资源泄漏检测分析...',
  '执行Shader错误分析...',
  '执行API反模式检测分析...',
  '执行性能异常定位分析...',
  '执行线程安全诊断分析...',
  '执行驱动层错误关联分析...',
  '执行小批量 DrawCall 检测分析...',
  '执行过量 glGetError 检测分析...',
  '生成诊断报告...',
]

async function runDiagnosis() {
  if (!props.filePath) {
    showToast('请先解析日志文件')
    return
  }
  diagnosing.value = true
  report.value = null

  let stepIdx = 0
  const progressTimer = setInterval(() => {
    if (stepIdx < progressSteps.length) {
      progressText.value = progressSteps[stepIdx]
      stepIdx++
    } else {
      progressText.value = '分析完成，正在整理结果...'
    }
  }, 800)

  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), 180000)

  try {
    const res = await fetch('/api/diagnose', {
      method: 'POST',
      signal: controller.signal,
    })
    clearTimeout(timeoutId)
    clearInterval(progressTimer)

    if (!res.ok) {
      const msg = await res.text()
      throw new Error(msg || `请求失败: ${res.status}`)
    }
    const data = await res.json()
    report.value = data as DiagnosisReport
    progressText.value = ''
    selectedSeverity.value = 'key'
    showAdvisory.value = false
    activeFinding.value = null
  } catch (err) {
    clearTimeout(timeoutId)
    clearInterval(progressTimer)
    if (err instanceof DOMException && err.name === 'AbortError') {
      showToast('诊断超时（180秒），文件可能过大')
    } else {
      showToast(err instanceof Error ? err.message : String(err))
    }
    progressText.value = ''
  } finally {
    diagnosing.value = false
  }
}

function toggleFinding(idx: number) {
  activeFinding.value = activeFinding.value === idx ? null : idx
}

function exportMarkdown() {
  if (!markdownReport.value) return
  const blob = new Blob([markdownReport.value], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `gpu_diagnosis_report.md`
  a.click()
  URL.revokeObjectURL(url)
  showToast('报告已导出', 'success')
}
</script>

<template>
  <div class="diagnosis-panel">
    <!-- Header -->
    <div class="panel-header">
      <h3>Bug 诊断</h3>
      <button
        class="btn btn-primary"
        :disabled="!filePath || diagnosing"
        @click="runDiagnosis"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        {{ diagnosing ? '诊断中...' : '开始诊断' }}
      </button>
    </div>

    <!-- Progress -->
    <div v-if="diagnosing" class="diagnosis-progress">
      <div class="spinner"></div>
      <p>{{ progressText || '准备中...' }}</p>
      <div class="progress-bar-wrap">
        <div class="progress-bar-fill" :style="{ width: Math.min((progressText ? progressSteps.indexOf(progressText) + 1 : 0) / progressSteps.length * 100, 100) + '%' }"></div>
      </div>
    </div>

    <!-- Report Summary -->
    <div v-if="report" class="diagnosis-report">
      <div class="report-summary">
        <div class="summary-stat critical">
          <span class="summary-count">{{ report.summary.critical_count }}</span>
          <span class="summary-label">严重</span>
        </div>
        <div class="summary-stat high">
          <span class="summary-count">{{ report.summary.high_count }}</span>
          <span class="summary-label">高</span>
        </div>
        <div class="summary-stat medium">
          <span class="summary-count">{{ report.summary.medium_count }}</span>
          <span class="summary-label">中</span>
        </div>
        <div class="summary-stat low">
          <span class="summary-count">{{ report.summary.low_count }}</span>
          <span class="summary-label">低</span>
        </div>
        <div class="summary-stat info">
          <span class="summary-count">{{ report.summary.info_count }}</span>
          <span class="summary-label">信息</span>
        </div>
        <div class="summary-stat total">
          <span class="summary-count">{{ report.summary.total_findings }}</span>
          <span class="summary-label">总计</span>
        </div>
      </div>

      <div class="report-meta">
        <span>源文件: {{ report.source_file }}</span>
        <span>生成时间: {{ report.generated_at }}</span>
      </div>

      <!-- Severity Filter -->
      <div class="severity-filter">
        <button
          :class="['filter-btn', { active: selectedSeverity === 'key' }]"
          @click="selectedSeverity = 'key'"
        >关键问题 ({{ keyFindingCount }})</button>
        <button
          :class="['filter-btn', { active: selectedSeverity === 'all' }]"
          @click="selectedSeverity = 'all'"
        >全部 ({{ report.summary.total_findings }})</button>
        <button
          v-if="report.summary.critical_count > 0"
          :class="['filter-btn', 'critical', { active: selectedSeverity === 'critical' }]"
          @click="selectedSeverity = 'critical'"
        >严重 ({{ report.summary.critical_count }})</button>
        <button
          v-if="report.summary.high_count > 0"
          :class="['filter-btn', 'high', { active: selectedSeverity === 'high' }]"
          @click="selectedSeverity = 'high'"
        >高 ({{ report.summary.high_count }})</button>
        <button
          v-if="report.summary.medium_count > 0"
          :class="['filter-btn', 'medium', { active: selectedSeverity === 'medium' }]"
          @click="selectedSeverity = 'medium'"
        >中 ({{ report.summary.medium_count }})</button>
        <button
          v-if="report.summary.low_count > 0"
          :class="['filter-btn', 'low', { active: selectedSeverity === 'low' }]"
          @click="selectedSeverity = 'low'"
        >低 ({{ report.summary.low_count }})</button>
        <button
          v-if="report.summary.info_count > 0"
          :class="['filter-btn', 'info', { active: selectedSeverity === 'info' }]"
          @click="selectedSeverity = 'info'"
        >信息 ({{ report.summary.info_count }})</button>
        <label v-if="advisoryFindingCount > 0" class="advisory-toggle">
          <input type="checkbox" v-model="showAdvisory">
          <span>显示低优先级建议 ({{ advisoryFindingCount }})</span>
        </label>
      </div>

      <!-- Findings -->
      <div class="findings-list">
        <div v-if="visibleFindings.length === 0" class="empty-state" style="padding:2rem;">
          <p>该级别无发现问题</p>
        </div>
        <div
          v-for="(finding, idx) in visibleFindings"
          :key="idx"
          class="finding-card"
          :class="{ expanded: activeFinding === idx }"
        >
          <div class="finding-header" @click="toggleFinding(idx)">
            <span
              class="severity-badge"
              :style="{ background: severityBgs[finding.severity], color: severityColors[finding.severity], borderColor: severityColors[finding.severity] }"
            >{{ severityLabels[finding.severity] || finding.severity }}</span>
            <span class="finding-desc">{{ finding.description }}</span>
            <span v-if="finding.category || finding.category_label" class="finding-category">{{ finding.category_label || finding.category }}</span>
            <span v-if="finding.count" class="finding-count">{{ finding.count }}x</span>
            <svg class="expand-icon" :class="{ rotated: activeFinding === idx }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
              <polyline points="6 9 12 15 18 9"/>
            </svg>
          </div>
          <div v-if="activeFinding === idx" class="finding-body">
            <div v-if="finding.kind || finding.confidence" class="finding-meta-row">
              <span v-if="finding.kind">类型: {{ finding.kind }}</span>
              <span v-if="finding.confidence">置信度: {{ finding.confidence }}</span>
            </div>
            <div v-if="finding.evidence" class="finding-section">
              <h4>证据</h4>
              <pre class="evidence-block">{{ finding.evidence }}</pre>
            </div>
            <div v-if="finding.examples && finding.examples.length > 0" class="finding-section">
              <h4>示例</h4>
              <ul class="finding-examples">
                <li v-for="example in finding.examples" :key="example">{{ example }}</li>
              </ul>
            </div>
            <div v-if="finding.root_cause_chain && finding.root_cause_chain.length > 0" class="finding-section">
              <h4>根因链</h4>
              <div class="root-cause-chain">
                <div v-for="(rc, rci) in finding.root_cause_chain" :key="rci" class="chain-node">
                  <div class="chain-index">{{ rci + 1 }}</div>
                  <div class="chain-content">{{ rc }}</div>
                  <div v-if="rci < finding.root_cause_chain.length - 1" class="chain-arrow">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                      <line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/>
                    </svg>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="finding.fix_suggestion" class="finding-section">
              <h4>修复建议</h4>
              <p class="fix-suggestion">{{ finding.fix_suggestion }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Export -->
      <div class="export-actions">
        <button class="btn btn-primary" @click="exportMarkdown">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          导出 Markdown 报告
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!report && !diagnosing" class="empty-state" style="padding:3rem;">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="40" height="40">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" y1="8" x2="12" y2="12"/>
        <line x1="12" y1="16" x2="12.01" y2="16"/>
      </svg>
      <p>还没有运行诊断</p>
      <small>点击"开始诊断"运行 Bug 分析器，默认优先展示高置信问题</small>
    </div>

    <!-- Toast -->
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
      <div class="toast-body"><p>{{ toast.message }}</p></div>
    </div>
  </div>
</template>

<style scoped>
.diagnosis-panel {
  min-height: 420px;
}

.diagnosis-progress {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 3rem 1rem;
}

.diagnosis-progress p {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.progress-bar-wrap {
  width: 100%;
  max-width: 480px;
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  background: var(--primary);
  border-radius: 2px;
  transition: width 0.4s ease;
}

.report-summary {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 0.75rem;
  margin-bottom: 1rem;
}

@media (max-width: 640px) {
  .report-summary {
    grid-template-columns: repeat(3, 1fr);
  }
}

.summary-stat {
  text-align: center;
  padding: 1rem 0.5rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-card);
}

.summary-stat.critical { border-left: 3px solid var(--danger); }
.summary-stat.high { border-left: 3px solid #fa8c16; }
.summary-stat.medium { border-left: 3px solid var(--warning); }
.summary-stat.low { border-left: 3px solid var(--primary); }
.summary-stat.info { border-left: 3px solid var(--text-secondary); }
.summary-stat.total { border-left: 3px solid var(--text-primary); }

.summary-count {
  display: block;
  font-size: 1.375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.summary-stat.critical .summary-count { color: var(--danger); }
.summary-stat.high .summary-count { color: #fa8c16; }
.summary-stat.medium .summary-count { color: var(--warning); }
.summary-stat.low .summary-count { color: var(--primary); }
.summary-stat.info .summary-count { color: var(--text-secondary); }
.summary-stat.total .summary-count { color: var(--text-primary); }

.summary-label {
  display: block;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-top: 0.25rem;
}

.report-meta {
  display: flex;
  gap: 1.5rem;
  font-size: 0.75rem;
  color: var(--text-placeholder);
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border);
}

.severity-filter {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1.25rem;
  align-items: center;
}

.filter-btn {
  padding: 0.375rem 0.875rem;
  border: 1px solid var(--border);
  border-radius: 20px;
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-family: var(--font);
  cursor: pointer;
  transition: all 0.15s;
}

.filter-btn:hover {
  border-color: var(--primary-border);
  color: var(--primary);
}

.filter-btn.active {
  background: var(--primary-light);
  border-color: var(--primary);
  color: var(--primary);
  font-weight: 600;
}

.filter-btn.critical { border-color: var(--danger-border); color: var(--danger); }
.filter-btn.critical.active { background: var(--danger-light); border-color: var(--danger); }
.filter-btn.high { border-color: #ffd591; color: #fa8c16; }
.filter-btn.high.active { background: #fff7e6; border-color: #fa8c16; }
.filter-btn.medium { border-color: #ffe58f; color: var(--warning); }
.filter-btn.medium.active { background: #fffbe6; border-color: var(--warning); }

.advisory-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  padding: 0.375rem 0.5rem;
}

.findings-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.finding-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: box-shadow 0.15s;
}

.finding-card:hover {
  box-shadow: var(--shadow-sm);
}

.finding-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  cursor: pointer;
  transition: background 0.1s;
}

.finding-header:hover {
  background: var(--bg-hover);
}

.severity-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.625rem;
  border-radius: 10px;
  border: 1px solid;
  white-space: nowrap;
  flex-shrink: 0;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.finding-desc {
  flex: 1;
  font-size: 0.875rem;
  color: var(--text-primary);
  line-height: 1.5;
}

.finding-category {
  font-size: 0.6875rem;
  color: var(--text-placeholder);
  background: var(--bg-page);
  padding: 0.125rem 0.5rem;
  border-radius: 8px;
  white-space: nowrap;
  flex-shrink: 0;
}

.finding-count {
  font-size: 0.6875rem;
  color: var(--primary);
  background: var(--primary-light);
  padding: 0.125rem 0.5rem;
  border-radius: 8px;
  white-space: nowrap;
  flex-shrink: 0;
}

.expand-icon {
  flex-shrink: 0;
  color: var(--text-placeholder);
  transition: transform 0.2s;
}

.expand-icon.rotated {
  transform: rotate(180deg);
}

.finding-body {
  padding: 0 1rem 1rem 1rem;
  animation: fadeIn 0.15s ease;
}

.finding-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.875rem;
}

.finding-meta-row span {
  font-size: 0.75rem;
  color: var(--text-secondary);
  background: var(--bg-page);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 0.125rem 0.625rem;
}

.finding-examples {
  margin-left: 1.25rem;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  line-height: 1.6;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.finding-section {
  margin-top: 0.875rem;
}

.finding-section h4 {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.5rem;
}

.evidence-block {
  background: var(--bg-page);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-primary);
  max-height: 200px;
  overflow-y: auto;
}

.root-cause-chain {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.chain-node {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.chain-index {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--primary-light);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  flex-shrink: 0;
  margin-top: 1px;
}

.chain-content {
  flex: 1;
  font-size: 0.8125rem;
  color: var(--text-primary);
  line-height: 1.6;
  padding-top: 4px;
}

.chain-arrow {
  width: 24px;
  display: flex;
  justify-content: center;
  color: var(--primary);
  opacity: 0.5;
  padding: 0.375rem 0;
}

.fix-suggestion {
  font-size: 0.8125rem;
  color: var(--text-primary);
  line-height: 1.6;
  padding: 0.75rem;
  background: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: var(--radius-sm);
}

.export-actions {
  margin-top: 1.5rem;
  padding-top: 1.25rem;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: flex-end;
}
</style>
