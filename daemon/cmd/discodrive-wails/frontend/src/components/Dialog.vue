<script setup>
import { watch } from 'vue'
import { X } from 'lucide-vue-next'
const props = defineProps({ open: Boolean, title: { type: String, default: '' } })
const emit = defineEmits(['close'])

function onKey(e) { if (e.key === 'Escape') emit('close') }
watch(() => props.open, (v) => {
  if (v) window.addEventListener('keydown', onKey)
  else window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close')">
    <div class="card w-full max-w-sm p-4">
      <div class="mb-3 flex items-center">
        <div class="text-sm font-semibold text-ink">{{ title }}</div>
        <button class="btn-ghost ml-auto" @click="emit('close')"><X :size="14" /></button>
      </div>
      <slot />
      <div class="mt-4 flex justify-end gap-2">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>
