<script setup>
import { ChevronRight } from 'lucide-vue-next'
defineProps({ frames: { type: Array, default: () => [] } })
const emit = defineEmits(['go'])
</script>

<template>
  <nav class="flex min-w-0 flex-wrap items-center gap-0.5 text-sm">
    <template v-for="(f, i) in frames" :key="i">
      <ChevronRight v-if="i > 0" :size="14" class="shrink-0 text-muted" />
      <button
        class="truncate rounded px-1.5 py-0.5 text-muted hover:bg-ink/5 hover:text-ink"
        :class="i === frames.length - 1 ? 'font-medium text-ink' : ''"
        @click="emit('go', i)"
      >
        {{ f.name }}
      </button>
    </template>
  </nav>
</template>
