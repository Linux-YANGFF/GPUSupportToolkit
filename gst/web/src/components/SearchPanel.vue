<script setup lang="ts">
import { inject } from 'vue'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const { searchKeyword, searchResults, searchTotal, searching, searched } = ctx
const { searchCurrentPage, searchTotalPages, paginatedSearchResults, searchPageRange } = ctx
const { doSearch, goSearchPage } = ctx
</script>

<template>
  <div>
    <div class="search-bar">
      <input
        type="text"
        v-model="searchKeyword"
        placeholder="输入搜索关键字..."
        class="input"
        @keyup.enter="doSearch"
      >
      <button class="btn btn-primary" @click="doSearch" :disabled="!searchKeyword || searching">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        搜索
      </button>
    </div>

    <div v-if="searchResults.length > 0" class="search-results-list">
      <div v-for="(result, idx) in paginatedSearchResults" :key="idx" class="search-result-item">
        <span class="result-line-no">[{{ result.line_number || (searchCurrentPage - 1) * 20 + idx + 1 }}]</span>
        <span class="result-text">{{ result.content }}</span>
      </div>
    </div>

    <div v-if="searchResults.length > 0" class="pagination">
      <div class="pagination-info">
        第 {{ searchCurrentPage }} / {{ searchTotalPages }} 页，共 {{ searchTotal.toLocaleString() }} 条结果
      </div>
      <div class="pagination-controls">
        <button class="page-btn" @click="goSearchPage(searchCurrentPage - 1)" :disabled="searchCurrentPage <= 1" title="上一页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>
        <template v-for="p in searchPageRange" :key="p">
          <button v-if="p !== '...'" class="page-btn" :class="{ active: p === searchCurrentPage }" @click="goSearchPage(p)">{{ p }}</button>
          <span v-else style="padding:0 0.25rem;color:var(--text-placeholder);">...</span>
        </template>
        <button class="page-btn" @click="goSearchPage(searchCurrentPage + 1)" :disabled="searchCurrentPage >= searchTotalPages" title="下一页">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
      </div>
    </div>

    <div v-if="searched && searchResults.length === 0" class="empty-state">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/>
        <line x1="12" y1="16" x2="12.01" y2="16"/>
      </svg>
      <p>未找到匹配结果</p>
      <small>尝试其他关键字</small>
    </div>

    <div v-if="!searched" class="empty-state">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
      <p>输入关键字进行搜索</p>
      <small>支持 API 名称、函数名、错误信息等</small>
    </div>
  </div>
</template>
