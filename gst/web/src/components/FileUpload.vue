<script setup lang="ts">
defineProps<{
  filePath: string
  parsing: boolean
  stopped: boolean
}>()

defineEmits<{
  'update:filePath': [value: string]
  browse: []
  parse: []
  reset: []
  stop: []
}>()
</script>

<template>
  <div class="card">
    <div class="file-row">
      <input
        type="text"
        :value="filePath"
        placeholder="输入日志文件路径..."
        class="input"
        @input="$emit('update:filePath', ($event.target as HTMLInputElement).value)"
        @keyup.enter="$emit('parse')"
      >
      <button class="btn btn-default" @click="$emit('browse')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        选择文件
      </button>
      <button class="btn btn-primary" @click="$emit('parse')" :disabled="parsing || (!filePath && !filePath)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/>
          <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
        </svg>
        解析
      </button>
      <button class="btn btn-danger" @click="$emit('reset')" :disabled="parsing">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
        重置
      </button>
      <button class="btn btn-danger" @click="$emit('stop')" :disabled="stopped">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="6" y="6" width="12" height="12"/>
        </svg>
        停止服务
      </button>
    </div>
    <div v-if="filePath && !filePath.startsWith('/')" style="margin-top:0.75rem;font-size:0.8125rem;color:var(--text-secondary);">
      已选择文件: <strong>{{ filePath }}</strong>
    </div>
  </div>
</template>
