<script setup lang="ts">
import { computed } from 'vue'
import { helpBlocks } from '../help-content'
import DebianSources from './DebianSources.vue'
import MarkdownContent from './MarkdownContent.vue'

const props = defineProps<{ source: string }>()
const blocks = computed(() => helpBlocks(props.source))
</script>

<template>
  <div class="mirror-help-content" lang="zh-CN">
    <template v-for="(block, index) in blocks" :key="index">
      <DebianSources v-if="block.kind === 'debian'" :format="block.format" />
      <MarkdownContent v-else :tokens="block.tokens" />
    </template>
  </div>
</template>
