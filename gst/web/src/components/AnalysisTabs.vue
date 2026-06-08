<script setup lang="ts">
import { inject } from 'vue'

const ctx = inject<any>('ctx')
if (!ctx) throw new Error('Missing provide ctx')

const { activeTab, tabs, switchTab } = ctx
</script>

<template>
  <div class="tabs-wrap">
    <div class="tabs-nav">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="switchTab(tab.id)"
        :class="['tab-btn', { active: activeTab === tab.id }]"
      >
        <svg v-if="tab.id === 'frames'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/>
        </svg>
        <svg v-else-if="tab.id === 'search'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <svg v-else-if="tab.id === 'analyze'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/>
        </svg>
        <svg v-else-if="tab.id === 'trace'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/>
          <circle cx="8" cy="5" r="2"/><circle cx="14" cy="12" r="2"/><circle cx="10" cy="19" r="2"/>
        </svg>
        <svg v-else-if="tab.id === 'export'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
        {{ tab.name }}
      </button>
    </div>

    <div class="tab-content">
      <div v-show="activeTab === 'frames'" class="tab-panel">
        <slot name="frames" />
      </div>
      <div v-show="activeTab === 'search'" class="tab-panel">
        <slot name="search" />
      </div>
      <div v-show="activeTab === 'analyze'" class="tab-panel">
        <slot name="analyze" />
      </div>
      <div v-show="activeTab === 'trace'" class="tab-panel">
        <slot name="trace" />
      </div>
      <div v-show="activeTab === 'export'" class="tab-panel">
        <slot name="export" />
      </div>
    </div>
  </div>
</template>
