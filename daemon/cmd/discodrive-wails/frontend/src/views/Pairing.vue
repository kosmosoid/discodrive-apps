<script setup>
import { ref } from 'vue'
import { Link, Loader2 } from 'lucide-vue-next'
import { api } from '../lib/api.js'
import { t } from '../lib/i18n.js'

const emit = defineEmits(['paired'])

const server = ref('https://')
const status = ref('')
const error = ref('')
const busy = ref(false)
const pairing = ref(null) // {userCode, verifyUrl, deviceCode, interval}

async function pair() {
  const url = server.value.trim()
  if (!url || url === 'https://') {
    error.value = t('pairing.needUrl')
    return
  }
  busy.value = true
  error.value = ''
  status.value = t('pairing.contacting')
  try {
    pairing.value = await api.pairInit(url)
    status.value = t('pairing.approve')
    await api.pairPoll(url, pairing.value.deviceCode, pairing.value.interval)
    emit('paired')
  } catch (e) {
    error.value = String(e)
    pairing.value = null
    status.value = ''
  } finally {
    busy.value = false
  }
}

function reopenLink() {
  if (pairing.value) api.openPairURL(pairing.value.verifyUrl)
}
</script>

<template>
  <div class="card mx-auto mt-16 max-w-md p-6">
    <div class="mb-1 text-base font-semibold text-ink">{{ t('pairing.title') }}</div>
    <p class="mb-4 text-xs text-muted">{{ t('pairing.subtitle') }}</p>

    <label class="mb-1 block text-xs text-muted">{{ t('pairing.serverUrl') }}</label>
    <input
      v-model="server"
      class="w-full rounded border border-line bg-panel2 px-2 py-1.5 text-sm text-ink outline-none focus:border-accent"
      placeholder="https://your-server.example.com"
      :disabled="busy"
      @keyup.enter="pair"
    />

    <button class="btn-accent mt-3 w-full justify-center" :disabled="busy" @click="pair">
      <Loader2 v-if="busy" :size="14" class="animate-spin" />
      {{ busy ? t('pairing.pairing') : t('pairing.pair') }}
    </button>

    <div v-if="pairing" class="mt-5 rounded-lg border border-line bg-panel2/60 p-4 text-center">
      <div class="text-xs text-muted">{{ t('pairing.enterCode') }}</div>
      <div class="my-1 font-mono text-2xl font-bold tracking-widest text-accent">{{ pairing.userCode }}</div>
      <button class="btn-ghost mx-auto mt-1" @click="reopenLink">
        <Link :size="13" /> {{ t('pairing.openLink') }}
      </button>
    </div>

    <p v-if="status" class="mt-4 text-center text-xs text-muted">{{ status }}</p>
    <p v-if="error" class="mt-4 rounded border border-danger/30 bg-danger/10 px-3 py-2 text-center text-xs text-danger">{{ error }}</p>
  </div>
</template>
