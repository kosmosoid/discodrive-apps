<script setup>
import { Folder, FolderOpen, ChevronRight, ChevronDown } from 'lucide-vue-next'
defineProps({
  node: { type: Object, required: true },
  selectedId: { type: String, default: '' },
})
const emit = defineEmits(['toggle', 'select'])
</script>

<template>
  <div>
    <div class="flex items-center">
      <button class="shrink-0 px-1 text-muted" @click="emit('toggle', node)">
        <component :is="node.expanded ? ChevronDown : ChevronRight" :size="13" />
      </button>
      <button
        class="flex min-w-0 flex-1 items-center gap-1.5 rounded px-1.5 py-1 text-left"
        :class="selectedId === node.id ? 'bg-accent/15 text-accent' : 'text-ink hover:bg-ink/5'"
        @click="emit('select', node)"
      >
        <component :is="node.expanded ? FolderOpen : Folder" :size="15" class="shrink-0 text-accent" />
        <span class="min-w-0 flex-1 truncate">{{ node.name }}</span>
      </button>
    </div>
    <div v-if="node.expanded && node.children && node.children.length" class="ml-4">
      <FolderPickerNode
        v-for="c in node.children"
        :key="c.id"
        :node="c"
        :selected-id="selectedId"
        @toggle="emit('toggle', $event)"
        @select="emit('select', $event)"
      />
    </div>
  </div>
</template>
