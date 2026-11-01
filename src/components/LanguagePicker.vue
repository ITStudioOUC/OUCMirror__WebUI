<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check } from 'lucide-vue-next'
import { supportedLocales, type Locale } from '../locales'

const emit = defineEmits<{ open: [] }>()
const { t, locale } = useI18n({ useScope: 'global' })
const open = ref(false)
const root = ref<HTMLElement>()
const trigger = ref<HTMLButtonElement>()
const menu = ref<HTMLElement>()

function languageName(value: string) {
  return t('common.languageName', {}, { locale: value })
}
function focusOption(index: number) {
  menu.value?.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]')[index]?.focus()
}
async function openMenu() {
  open.value = true
  emit('open')
  await nextTick()
  if (open.value) focusOption(Math.max(0, supportedLocales.indexOf(locale.value as Locale)))
}
function closeMenu(restoreFocus = false) {
  open.value = false
  if (restoreFocus) trigger.value?.focus()
}
function selectLanguage(value: Locale) {
  locale.value = value
  closeMenu(true)
}
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && open.value) {
    event.preventDefault()
    event.stopPropagation()
    closeMenu(true)
  } else if (event.key === 'Tab' && open.value) {
    // Continue normal tab order from the trigger after dismissing the menu.
    closeMenu(true)
  } else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    if (!open.value) { void openMenu(); return }
    const options = [...(menu.value?.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]') || [])]
    const current = options.indexOf(document.activeElement as HTMLButtonElement)
    const next = current < 0 ? 0 : (current + (event.key === 'ArrowDown' ? 1 : -1) + options.length) % options.length
    focusOption(next)
  } else if (open.value && (event.key === 'Home' || event.key === 'End')) {
    event.preventDefault()
    focusOption(event.key === 'Home' ? 0 : supportedLocales.length - 1)
  }
}
function onFocusOut(event: FocusEvent) {
  if (!root.value?.contains(event.relatedTarget as Node | null)) closeMenu()
}
function onOutsidePointer(event: PointerEvent) {
  if (!root.value?.contains(event.target as Node)) closeMenu()
}
onMounted(() => document.addEventListener('pointerdown', onOutsidePointer))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutsidePointer))
</script>

<template>
  <div ref="root" class="language-picker" @keydown="onKeydown" @focusout="onFocusOut">
    <button ref="trigger" type="button" class="language-trigger" aria-haspopup="menu" :aria-expanded="open" aria-controls="language-options" :aria-label="`${t('common.language')}: ${languageName(locale)}`" @click="open ? closeMenu() : openMenu()">
      {{ languageName(locale) }}
    </button>
    <div v-if="open" id="language-options" ref="menu" class="language-options" role="menu" :aria-label="t('common.language')">
      <button v-for="choice in supportedLocales" :key="choice" type="button" class="language-option" role="menuitemradio" :aria-checked="locale === choice" :lang="choice" tabindex="-1" @click="selectLanguage(choice)">
        <span>{{ languageName(choice) }}</span>
        <Check v-if="locale === choice" :size="15" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>
