<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  value: string
  cycle?: number
  candidates?: string[]
  reserve?: string[]
  shuffle?: boolean
}>(), {
  cycle: 0,
  candidates: () => [],
  reserve: () => [],
  shuffle: true,
})

const shown = ref(props.value)
const incoming = ref(props.value)
const rolling = ref(false)
const busy = ref(false)
const revision = ref(0)
const duration = ref(140)
const reservations = computed(() => [...new Set([props.value, ...props.reserve])])
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
let timer: ReturnType<typeof setTimeout> | undefined

function settle() {
  clearTimeout(timer)
  timer = undefined
  rolling.value = false
  busy.value = false
  shown.value = props.value
  incoming.value = props.value
}

function spin() {
  clearTimeout(timer)
  rolling.value = false
  if (reducedMotion.matches || document.hidden) {
    settle()
    return
  }
  busy.value = true

  // A shared cycle starts the whole board; each readout has its own mechanical cadence.
  const target = props.value
  let remaining = props.shuffle ? 3 + Math.floor(Math.random() * 4) : 1

  function roll() {
    const pool = props.candidates.filter(value => value !== shown.value)
    incoming.value = remaining === 1 ? target
      : pool[Math.floor(Math.random() * pool.length)] ?? target
    duration.value = props.shuffle ? 110 + Math.floor(Math.random() * 41) : 220
    rolling.value = true
    revision.value++
    timer = setTimeout(() => {
      shown.value = incoming.value
      rolling.value = false
      if (--remaining > 0) timer = setTimeout(roll, 16)
      else { timer = undefined; busy.value = false }
    }, duration.value + 20)
  }

  timer = setTimeout(roll, props.shuffle ? Math.floor(Math.random() * 90) : 0)
}

function onVisibilityChange() {
  if (document.hidden) settle()
}

watch([() => props.value, () => props.cycle], spin)
onMounted(() => {
  reducedMotion.addEventListener('change', settle)
  document.addEventListener('visibilitychange', onVisibilityChange)
  spin()
})
onBeforeUnmount(() => {
  clearTimeout(timer)
  reducedMotion.removeEventListener('change', settle)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <span class="roll-frame" :data-rolling="rolling" :data-busy="busy">
    <span class="sr-only">{{ value }}</span>
    <span class="roll-stage" aria-hidden="true" :data-roll="revision" :style="{ '--roll-duration': `${duration}ms` }">
      <span v-for="text in reservations" :key="text" class="roll-sizer">{{ text }}</span>
      <span v-if="rolling" :key="revision" class="roll-track">
        <span class="roll-face">{{ shown }}</span>
        <span class="roll-face">{{ incoming }}</span>
      </span>
      <span v-else class="roll-rest">{{ shown }}</span>
    </span>
  </span>
</template>
