<script setup lang="ts">
import { computed, reactive, ref, useId, watch, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import { configDefaults, configFields, generateMirrorConfig, getMirrorGenerator, parseMirrorConfig, type ConfigChoice, type ConfigValues, type MirrorConfig } from '../mirror-config'
import { defaultGeneratorCatalog, type GeneratorCatalog } from '../generator-catalog'
import ConfigCode from './ConfigCode.vue'

const props = withDefaults(defineProps<{ config: MirrorConfig; catalog?: GeneratorCatalog }>(), { catalog: () => defaultGeneratorCatalog })
const { t } = useI18n()
const id = useId()
const values = reactive<ConfigValues>({})
const format = ref('')
const generator = computed(() => getMirrorGenerator(props.config.type, props.catalog))
const fields = computed(() => configFields(props.config.type, props.catalog))
watch(() => JSON.stringify(props.config), () => {
  for (const key of Object.keys(values)) delete values[key]
  Object.assign(values, configDefaults(props.config, window.location.origin, props.catalog))
  format.value = props.config.format || generator.value.defaultFormat
}, { immediate: true })
watch(() => props.catalog, () => {
  if (!parseMirrorConfig({ ...props.config, options: { ...props.config.options, ...values } }, props.catalog)) Object.assign(values, configDefaults(props.config, window.location.origin, props.catalog))
})
watchEffect(() => generator.value.normalize?.(values))
const files = computed(() => generateMirrorConfig({ ...props.config, format: format.value }, window.location.origin, values, props.catalog))
const notes = computed(() => (generator.value.notes || []).filter(note => !note.when || note.when(values)))
const choiceLabel = (choice: ConfigChoice) => choice.labelKey ? t(choice.labelKey) : choice.label
</script>

<template>
  <section class="mirror-config" :data-generator="config.type" :data-format="format" :aria-label="t('mirrors.generator.title', { name: t(generator.nameKey) })">
    <div class="source-controls">
      <label v-if="!config.format && generator.formats.length > 1" class="source-control"><span>{{ t('mirrors.generator.format') }}</span><select v-model="format"><option v-for="choice in generator.formats" :key="choice.value" :value="choice.value">{{ choiceLabel(choice) }}</option></select></label>
      <label v-for="field in fields" :key="field.key" class="source-control" :class="{ 'source-disabled': field.disabled?.(values) }">
        <span>{{ t(field.labelKey) }}</span>
        <select v-if="field.kind === 'select'" v-model="values[field.key]"><option v-for="choice in field.choices" :key="choice.value" :value="choice.value">{{ choiceLabel(choice) }}</option></select>
        <template v-else><input v-model="values[field.key]" type="checkbox" role="switch" :disabled="field.disabled?.(values)" :aria-describedby="field.disabled?.(values) && notes.length ? `${id}-notes` : undefined" /><span class="source-toggle" aria-hidden="true" /></template>
      </label>
    </div>
    <div v-if="notes.length" :id="`${id}-notes`" class="source-notes"><p v-for="note in notes" :key="note.key">{{ t(note.key) }}</p></div>
    <ConfigCode v-for="(file, index) in files" :key="index" :code="file.code" :caption="file.path || t(file.titleKey!)" />
  </section>
</template>

<style scoped>
.mirror-config { min-width: 0; margin: 20px 0 30px; }
.source-controls { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 180px), 1fr)); gap: 12px; margin-bottom: 16px; }
.source-control { position: relative; display: flex; flex-direction: column; justify-content: space-between; align-items: flex-start; gap: 8px; min-width: 0; min-height: 70px; margin: 0; padding: 10px 12px; border: 1px solid var(--line); border-radius: 4px; color: var(--ink); font-size: 13px; line-height: 1.5; cursor: pointer; }
.source-control > span:first-child { color: var(--ink-2); }
.source-control select { width: 100%; min-width: 0; margin: 0; padding: 0; border: 0; background: var(--paper); color: var(--ink); font: inherit; font-size: 15px; cursor: pointer; }
.source-control input { position: absolute; width: 1px; height: 1px; margin: 0; opacity: 0; }
.source-toggle { position: relative; width: 32px; height: 18px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); }
.source-toggle::after { position: absolute; top: 3px; left: 3px; width: 10px; height: 10px; border-radius: 50%; background: var(--ink-2); content: ''; transition: transform 120ms; }
.source-control input:checked + .source-toggle { border-color: var(--blue); background: var(--blue); }
.source-control input:checked + .source-toggle::after { background: var(--on-accent); transform: translateX(14px); }
.source-control input:focus-visible + .source-toggle { outline: 2px solid var(--blue); outline-offset: 3px; }
.source-disabled { opacity: .55; cursor: default; }
.source-notes { margin: 10px 0; color: var(--ink); font-size: 14px; line-height: 1.7; }
.source-notes p { margin: 6px 0; }
@media (max-width: 640px) {
  .source-controls { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
  .source-control { min-height: 80px; padding: 10px; }
  .source-control select { min-height: 28px; font-size: 14px; }
}
@media (max-width: 360px) { .source-controls { grid-template-columns: minmax(0, 1fr); } }
@media (prefers-reduced-motion: reduce) { .source-toggle::after { transition: none; } }
</style>
