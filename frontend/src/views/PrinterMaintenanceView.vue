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
          <UButton color="warning" variant="outline" icon="i-lucide-droplets" :loading="running === 'clean'" :disabled="!!running" @click="openConfirm('clean')">
            {{ t('ui.normal_cleaning') }}
          </UButton>
          <UButton color="error" variant="outline" icon="i-lucide-droplets" :loading="running === 'deep-clean'" :disabled="!!running" @click="openConfirm('deep-clean')">
            {{ t('ui.deep_cleaning') }}
          </UButton>
          <UButton variant="outline" icon="i-lucide-rotate-cw" :loading="running === 'roller-clean'" :disabled="!!running" @click="openConfirm('roller-clean')">
            {{ t('ui.roller_cleaning') }}
          </UButton>
          <UButton variant="outline" icon="i-lucide-file-cleanup" :loading="running === 'bottom-plate-clean'" :disabled="!!running" @click="openConfirm('bottom-plate-clean')">
            {{ t('ui.bottom_plate_cleaning') }}
          </UButton>
        </div>
        <UAlert v-if="result" color="success" variant="subtle" icon="i-lucide-check-circle" :title="result" />
        <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-circle-x" :title="error" />
      </div>
    </UCard>

    <UModal :title="confirmTitle" v-model:open="confirmOpen">
      <template #content>
        <div class="p-6 space-y-4">
          <p>{{ confirmText }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="confirmOpen = false">{{ t('ui.cancel') }}</UButton>
            <UButton :color="pendingAction === 'deep-clean' ? 'error' : 'warning'" :loading="running === pendingAction" @click="run(pendingAction)">{{ confirmButton }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch, readError } from '../utils/api.js'

const { t } = useI18n()
const running = ref('')
const error = ref('')
const result = ref('')
const confirmOpen = ref(false)
const pendingAction = ref('')

const confirmationKeys = {
  clean: ['ui.confirm_normal_cleaning', 'ui.normal_cleaning_confirmation', 'ui.normal_cleaning'],
  'deep-clean': ['ui.confirm_deep_cleaning', 'ui.deep_cleaning_confirmation', 'ui.deep_cleaning'],
  'roller-clean': ['ui.confirm_roller_cleaning', 'ui.roller_cleaning_confirmation', 'ui.roller_cleaning'],
  'bottom-plate-clean': ['ui.confirm_bottom_plate_cleaning', 'ui.bottom_plate_cleaning_confirmation', 'ui.bottom_plate_cleaning']
}
const confirmTitle = computed(() => t(confirmationKeys[pendingAction.value]?.[0] || 'ui.printer_maintenance'))
const confirmText = computed(() => t(confirmationKeys[pendingAction.value]?.[1] || 'ui.printer_maintenance_description'))
const confirmButton = computed(() => t(confirmationKeys[pendingAction.value]?.[2] || 'ui.printer_maintenance'))

function openConfirm(action) {
  pendingAction.value = action
  confirmOpen.value = true
}

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
    confirmOpen.value = false
  } catch (e) {
    error.value = e.message || t('ui.maintenance_job_failed')
  } finally {
    running.value = ''
  }
}
</script>
