<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGeneratorCatalog } from '../useGeneratorCatalog'
import { helpBlocks } from '../help-content'
import MirrorConfig from './MirrorConfig.vue'
import MarkdownContent from './MarkdownContent.vue'

const props = defineProps<{ source: string }>()
const { t } = useI18n()
const { catalog, failed, refresh } = useGeneratorCatalog()
const blocks = computed(() => catalog.value ? helpBlocks(props.source, catalog.value) : [])
</script>

<template>
  <div class="mirror-help-content" lang="zh-CN">
    <p v-if="!catalog" role="status">{{ t(failed ? 'mirrors.generator.catalogFailed' : 'mirrors.guides.loading') }} <button v-if="failed" type="button" @click="refresh">{{ t('mirrors.guides.retry') }}</button></p>
    <p v-else-if="failed" role="status">{{ t('mirrors.generator.catalogStale') }}</p>
    <template v-for="(block, index) in blocks" :key="index">
      <MirrorConfig v-if="block.kind === 'config' && catalog" :config="block.config" :catalog="catalog" />
      <MarkdownContent v-else-if="block.kind === 'markdown'" :tokens="block.tokens" />
    </template>
  </div>
</template>
