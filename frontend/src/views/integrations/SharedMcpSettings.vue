<template>
  <div class="shared-mcp-panel">
    <t-alert theme="info" :message="sharedUrlHint" />
    <t-form v-if="isAdmin" label-align="top" class="shared-mcp-form">
      <t-form-item :label="$t('integrations.sharedMcp.enabled')">
        <t-switch v-model="form.enabled" :loading="saving" @change="save" />
      </t-form-item>
      <t-form-item :label="$t('integrations.sharedMcp.url')">
        <t-input :value="sharedMcpUrl" readonly>
          <template #suffix>
            <t-button size="small" variant="text" @click="copyUrl">{{ $t('common.copy') }}</t-button>
          </template>
        </t-input>
      </t-form-item>
    </t-form>
    <p v-else class="shared-mcp-readonly">{{ sharedMcpUrl }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { getSharedMcpConfig, updateSharedMcpConfig, type SharedMcpConfig } from '@/api/shared-mcp'
import { useApiBaseUrlDisplay } from '@/composables/useApiBaseUrlDisplay'
import { useAuthStore } from '@/stores/auth'
import { buildMcpEndpointUrl } from '@/views/integrations/mcpServerIntegration'

const { t } = useI18n()
const { apiBaseUrlDisplay } = useApiBaseUrlDisplay()
const authStore = useAuthStore()
const isAdmin = computed(() => authStore.hasRole('admin'))

const loading = ref(false)
const saving = ref(false)
const form = ref<SharedMcpConfig>({
  tenant_id: 0,
  enabled: false,
  tools: [],
  knowledge_base_ids: [],
  default_agent_id: '',
  rate_limit_per_minute: 60,
  redirect_uri_allowlist: [],
})

const sharedMcpUrl = computed(() => buildMcpEndpointUrl(apiBaseUrlDisplay.value, '/mcp'))

const sharedUrlHint = computed(() =>
  t('integrations.sharedMcp.hint', { url: sharedMcpUrl.value }),
)

async function load() {
  loading.value = true
  try {
    const res = await getSharedMcpConfig()
    if (res.data?.data) {
      form.value = res.data.data
    }
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const res = await updateSharedMcpConfig({
      enabled: form.value.enabled,
      tools: form.value.tools,
      knowledge_base_ids: form.value.knowledge_base_ids,
      default_agent_id: form.value.default_agent_id,
      rate_limit_per_minute: form.value.rate_limit_per_minute,
      redirect_uri_allowlist: form.value.redirect_uri_allowlist,
    })
    if (res.data?.data) {
      form.value = res.data.data
    }
  } finally {
    saving.value = false
  }
}

async function copyUrl() {
  try {
    await navigator.clipboard.writeText(sharedMcpUrl.value)
    MessagePlugin.success(t('common.copied'))
  } catch {
    MessagePlugin.error(t('common.copyFailed'))
  }
}

onMounted(load)
</script>

<style scoped>
.shared-mcp-panel {
  margin-bottom: 24px;
}
.shared-mcp-form {
  margin-top: 16px;
  max-width: 640px;
}
.shared-mcp-readonly {
  margin-top: 12px;
  word-break: break-all;
}
</style>
