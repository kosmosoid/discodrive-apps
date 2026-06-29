<script setup>
import { Lock, LockOpen, FolderOpen, FilePlus, Loader2 } from 'lucide-vue-next'
import { t } from '../lib/i18n.js'
defineProps({
  vault: { type: Object, required: true },
  busy: { type: Boolean, default: false },
})
const emit = defineEmits(['open', 'openFolder', 'addFiles', 'close'])
</script>

<template>
  <div class="group flex items-center gap-2 rounded px-2 py-2 hover:bg-ink/5">
    <component :is="vault.open ? LockOpen : Lock" :size="16" :class="vault.open ? 'shrink-0 text-accent2' : 'shrink-0 text-muted'" />
    <span class="min-w-0 flex-1 truncate text-sm text-ink">{{ vault.name }}</span>
    <span v-if="vault.open" class="shrink-0 text-xs text-accent2">{{ t('vaults.openState') }}</span>

    <Loader2 v-if="busy" :size="15" class="ml-1 shrink-0 animate-spin text-accent" />
    <div v-else class="ml-1 flex shrink-0 items-center gap-0.5">
      <template v-if="vault.open">
        <button class="btn-ghost px-1.5 py-1" :title="t('vaults.openFolder')" @click.stop="emit('openFolder', vault)"><FolderOpen :size="14" /></button>
        <button class="btn-ghost px-1.5 py-1" :title="t('vaults.addFiles')" @click.stop="emit('addFiles', vault)"><FilePlus :size="14" /></button>
        <button class="btn-accent px-2 py-1 text-xs" @click.stop="emit('close', vault)">{{ t('common.close') }}</button>
      </template>
      <button v-else class="btn-accent px-2 py-1 text-xs" @click.stop="emit('open', vault)">{{ t('common.open') }}</button>
    </div>
  </div>
</template>
