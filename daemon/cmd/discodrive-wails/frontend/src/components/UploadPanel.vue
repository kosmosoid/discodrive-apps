<script setup>
import { UploadCloud, Check, AlertCircle, Loader2, X } from 'lucide-vue-next'
import { t } from '../lib/i18n.js'
defineProps({ uploads: { type: Array, default: () => [] } })
defineEmits(['dismiss'])

const pct = (u) => (u.total > 0 ? Math.min(100, Math.round((u.sent / u.total) * 100)) : 0)
const done = (u) => u.state === 'error' || u.state === 'done'
</script>

<template>
  <div v-if="uploads.length" class="fixed bottom-4 right-4 z-40 w-72 card p-3">
    <div class="mb-2 flex items-center gap-1.5 text-xs font-semibold text-ink">
      <UploadCloud :size="14" class="text-accent" />
      {{ t('uploads.title') }}
    </div>
    <ul class="space-y-2">
      <li v-for="u in uploads" :key="u.id" class="text-xs">
        <div class="flex items-center gap-1.5">
          <Loader2 v-if="u.state === 'uploading'" :size="12" class="shrink-0 animate-spin text-accent" />
          <Check v-else-if="u.state === 'done'" :size="12" class="shrink-0 text-accent2" />
          <AlertCircle v-else-if="u.state === 'error'" :size="12" class="shrink-0 text-danger" />
          <span class="min-w-0 flex-1 truncate text-ink" :title="u.state === 'error' ? u.error : u.name">{{ u.name }}</span>
          <span class="shrink-0 text-muted">{{ u.state === 'error' ? t('uploads.failed') : pct(u) + '%' }}</span>
          <button
            v-if="done(u)"
            class="shrink-0 text-muted hover:text-ink"
            :title="t('uploads.dismiss')"
            @click="$emit('dismiss', u.id)"
          >
            <X :size="12" />
          </button>
        </div>
        <p
          v-if="u.state === 'error' && (u.error || u.code)"
          class="mt-0.5 truncate text-[11px] text-danger/80"
          :title="u.error"
        >
          {{ u.code === 'name_too_long' ? t('uploads.nameTooLong') : u.error }}
        </p>
        <div class="mt-1 h-1 overflow-hidden rounded bg-ink/10">
          <div
            class="h-full transition-all"
            :class="u.state === 'error' ? 'bg-danger' : u.state === 'done' ? 'bg-accent2' : 'bg-accent'"
            :style="{ width: (u.state === 'error' ? 100 : pct(u)) + '%' }"
          />
        </div>
      </li>
    </ul>
  </div>
</template>
