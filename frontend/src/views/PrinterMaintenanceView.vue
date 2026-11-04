<template>
  <div class="p-3 sm:p-4 md:p-6 max-w-2xl">
    <UCard>
      <template #header>
        <h2 class="text-xl font-bold flex items-center gap-2">
          <UIcon name="i-lucide-wrench" class="w-5 h-5" /> {{ t('ui.printer_maintenance') }}
        </h2>
      </template>
      <div class="space-y-4">
        <p class="text-sm text-muted">{{ t('ui.printer_maintenance_description') }}</p>
        <UAlert color="warning" variant="subtle" icon="i-lucide-triangle-alert" :title="t('ui.printer_maintenance_warning')" />
        <div class="flex flex-wrap gap-2">
          <UButton icon="i-lucide-printer-check" :loading="running === 'nozzle-check'" :disabled="!!running" @click="run('nozzle-check')">
            {{ t('ui.nozzle_check') }}
          </UButton>
          <UButton color="warning" variant="outline" icon="i-lucide-droplets" :loading="running === 'clean'" :disabled="!!running" @click="confirmCleaning = true">
            {{ t('ui.normal_cleaning') }}
          </UButton>
        </div>
        <UAlert v-if="result" color="success" variant="subtle" icon="i-lucide-check-circle" :title="result" />
        <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-circle-x" :title="error" />
      </div>
    </UCard>

    <UModal :title="t('ui.confirm_normal_cleaning')" v-model:open="confirmCleaning">
      <template #content>
        <div class="p-6 space-y-4">
          <p>{{ t('ui.normal_cleaning_confirmation') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="confirmCleaning = false">{{ t('ui.cancel') }}</UButton>
            <UButton color="warning" :loading="running === 'clean'" @click="run('clean')">{{ t('ui.normal_cleaning') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch, readError } from '../utils/api.js'

const { t } = useI18n()
const running = ref('')
const error = ref('')
const result = ref('')
const confirmCleaning = ref(false)

async function run(action) {
  running.value = action
  error.value = ''
  result.value = ''
  try {
    const resp = await apiFetch('/api/admin/printer-maintenance', {
      method: 'POST',
      body: JSON.stringify({ action })
    })
    if (!resp.ok) throw new Error(await readError(resp))
    const data = await resp.json()
    result.value = data.job || t('ui.maintenance_job_submitted')
    confirmCleaning.value = false
  } catch (e) {
    error.value = e.message || t('ui.maintenance_job_failed')
  } finally {
    running.value = ''
  }
}
</script>
