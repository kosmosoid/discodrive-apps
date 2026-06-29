<script setup>
import { ref, onMounted } from 'vue'
import { Sun, Moon, FolderOpen, Globe, ChevronDown } from 'lucide-vue-next'
import { api } from '../lib/api.js'
import { t, setLocale, languages } from '../lib/i18n.js'
import { applyTheme } from '../lib/theme.js'
import Dialog from '../components/Dialog.vue'

const emit = defineEmits(['unpaired'])

const settings = ref({ theme: 'dark', lang: 'en', openAtLogin: false, startMinimized: false })
const server = ref('')
const cache = ref('')
const confirmUnpair = ref(false)

onMounted(async () => {
  settings.value = (await api.getSettings()) || settings.value
  server.value = await api.serverURL()
  cache.value = await api.cachePath()
})

async function persist() {
  await api.saveSettings(settings.value)
}

function setTheme(theme) {
  settings.value.theme = theme
  applyTheme(theme)
  persist()
}
function setLang(lang) {
  settings.value.lang = lang
  setLocale(lang)
  persist()
}
function toggleOpenAtLogin() {
  settings.value.openAtLogin = !settings.value.openAtLogin
  persist()
}
function toggleStartMinimized() {
  settings.value.startMinimized = !settings.value.startMinimized
  persist()
}
async function doChangeServer() {
  confirmUnpair.value = false
  await api.unpair()
  emit('unpaired')
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex items-center gap-2 border-b border-line px-4 py-2">
      <div class="text-sm font-medium text-ink">{{ t('settings.title') }}</div>
    </div>

    <div class="min-h-0 flex-1 overflow-auto p-4">
      <div class="mx-auto max-w-md space-y-5">
        <!-- Theme -->
        <div>
          <div class="mb-1.5 text-xs font-medium text-muted">{{ t('settings.theme') }}</div>
          <div class="flex gap-2">
            <button
              class="btn flex-1 justify-center ring-1"
              :class="settings.theme === 'dark' ? 'bg-accent/15 text-accent ring-accent/30' : 'text-muted ring-line hover:text-ink'"
              @click="setTheme('dark')"
            >
              <Moon :size="14" /> {{ t('settings.dark') }}
            </button>
            <button
              class="btn flex-1 justify-center ring-1"
              :class="settings.theme === 'light' ? 'bg-accent/15 text-accent ring-accent/30' : 'text-muted ring-line hover:text-ink'"
              @click="setTheme('light')"
            >
              <Sun :size="14" /> {{ t('settings.light') }}
            </button>
          </div>
        </div>

        <!-- Language -->
        <div>
          <div class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-muted">
            <Globe :size="13" /> {{ t('settings.language') }}
          </div>
          <div class="relative">
            <select
              :value="settings.lang"
              class="w-full appearance-none rounded border border-line bg-panel2 px-2 py-1.5 pr-8 text-sm text-ink outline-none focus:border-accent"
              @change="setLang($event.target.value)"
            >
              <option v-for="l in languages" :key="l.code" :value="l.code">{{ l.label }}</option>
            </select>
            <ChevronDown
              :size="15"
              class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-muted"
            />
          </div>
        </div>

        <!-- Server -->
        <div>
          <div class="mb-1.5 text-xs font-medium text-muted">{{ t('settings.server') }}</div>
          <div class="flex items-center gap-2">
            <input :value="server" readonly class="min-w-0 flex-1 truncate rounded border border-line bg-panel2 px-2 py-1.5 text-sm text-ink outline-none" />
            <button class="btn-ghost shrink-0" @click="confirmUnpair = true">{{ t('settings.changeServer') }}</button>
          </div>
        </div>

        <!-- Cache -->
        <div>
          <div class="mb-1.5 text-xs font-medium text-muted">{{ t('settings.cache') }}</div>
          <div class="flex items-center gap-2">
            <input :value="cache" readonly class="min-w-0 flex-1 truncate rounded border border-line bg-panel2 px-2 py-1.5 text-xs text-ink outline-none" />
            <button class="btn-ghost shrink-0" @click="api.revealCache()"><FolderOpen :size="14" /> {{ t('settings.openCache') }}</button>
          </div>
        </div>

        <!-- Open at login -->
        <label class="flex cursor-pointer items-center gap-2">
          <input type="checkbox" :checked="settings.openAtLogin" class="accent-accent" @change="toggleOpenAtLogin" />
          <span class="text-sm text-ink">{{ t('settings.openAtLogin') }}</span>
        </label>

        <!-- Start minimized (only meaningful when launched at login) -->
        <label
          class="flex items-center gap-2 pl-6"
          :class="settings.openAtLogin ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'"
        >
          <input
            type="checkbox"
            :checked="settings.startMinimized"
            :disabled="!settings.openAtLogin"
            class="accent-accent"
            @change="toggleStartMinimized"
          />
          <span class="text-sm text-ink">{{ t('settings.startMinimized') }}</span>
        </label>
      </div>
    </div>

    <Dialog :open="confirmUnpair" :title="t('settings.changeServerTitle')" @close="confirmUnpair = false">
      <p class="text-sm text-muted">{{ t('settings.changeServerConfirm') }}</p>
      <template #footer>
        <button class="btn-ghost" @click="confirmUnpair = false">{{ t('common.cancel') }}</button>
        <button class="btn-accent !bg-danger/15 !text-danger !ring-danger/30" @click="doChangeServer">{{ t('settings.changeServer') }}</button>
      </template>
    </Dialog>
  </div>
</template>
