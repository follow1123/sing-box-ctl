<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import PageShell from '../components/PageShell.vue'
import { api } from '../composables/useApi'
import type { ProviderConfig, TemplateInfo } from '../types'

const providers = ref<ProviderConfig[]>([])
const templates = ref<TemplateInfo[]>([])
const error = ref('')

const providerUuid = ref('')
const templateUuid = ref('')
const platform = ref('windows')
const tun = ref(false)
const mixed = ref(false)
const mixedPort = ref('')
const sysProxy = ref(false)
const share = ref(false)
const webui = ref(false)
const webuiPort = ref('')
const webuiSecret = ref('')
const importUrlInput = ref('')

const url = computed(() => buildUrl())

function buildUrl(): string {
  if (!providerUuid.value) return ''
  const params: string[] = []
  if (templateUuid.value) params.push('template=' + encodeURIComponent(templateUuid.value))
  if (platform.value) params.push('platform=' + platform.value)
  if (tun.value) params.push('tun')
  if (mixed.value) {
    params.push('mixed')
    if (mixedPort.value) params.push('mixed-port=' + mixedPort.value)
    if (sysProxy.value) params.push('sys-proxy')
    if (share.value) params.push('share')
  }
  if (webui.value) {
    if (webuiPort.value) params.push('webui-port=' + webuiPort.value)
    if (webuiSecret.value) params.push('webui-secret=' + encodeURIComponent(webuiSecret.value))
  }
  return location.origin + '/config/' + providerUuid.value + (params.length ? '?' + params.join('&') : '')
}

function importFromUrl(raw: string): void {
  try {
    const u = new URL(raw, location.origin)
    const m = u.pathname.match(/\/config\/([^/]+)/)
    if (m) providerUuid.value = m[1]
    const q = u.searchParams
    if (q.has('template')) templateUuid.value = q.get('template') || ''
    if (q.has('platform')) platform.value = q.get('platform') || 'windows'
    tun.value = q.has('tun')
    mixed.value = q.has('mixed')
    mixedPort.value = q.get('mixed-port') || ''
    sysProxy.value = q.has('sys-proxy')
    share.value = q.has('share')
    webui.value = q.has('webui-port') || q.has('webui-secret')
    webuiPort.value = q.get('webui-port') || ''
    webuiSecret.value = q.get('webui-secret') || ''
  } catch (e) {
    error.value = '导入失败: ' + (e as Error).message
  }
}

async function copyUrl(): Promise<void> {
  if (!providerUuid.value) {
    error.value = '请先选择 provider'
    return
  }
  try {
    await navigator.clipboard.writeText(url.value)
    error.value = ''
  } catch {
    // 旧浏览器降级
    const ta = document.createElement('textarea')
    ta.value = url.value
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
}

function openUrl(): void {
  if (!providerUuid.value) {
    error.value = '请先选择 provider'
    return
  }
  window.open(url.value, '_blank')
}

onMounted(async () => {
  try {
    const [ps, ts] = await Promise.all([
      api<ProviderConfig[]>('/api/providers'),
      api<TemplateInfo[]>('/api/templates'),
    ])
    providers.value = ps
    templates.value = ts
    const def = ts.find((t) => t.default)
    if (def) templateUuid.value = def.uuid
  } catch (e) {
    error.value = '加载数据失败: ' + (e as Error).message
  }
})
</script>

<template>
  <PageShell>
    <template #aside>
      <h2>从 URL 导入</h2>
      <input v-model="importUrlInput" type="text" placeholder="粘贴已有 URL" />
      <button class="import-btn" @click="importFromUrl(importUrlInput)">导入</button>

      <h2 class="group-title">选择</h2>
      <div class="field">
        <label>Provider</label>
        <select v-model="providerUuid">
          <option value="">-- 选择 provider --</option>
          <option v-for="p in providers" :key="p.uuid" :value="p.uuid">{{ p.name }}</option>
        </select>
      </div>
      <div class="field">
        <label>模板</label>
        <select v-model="templateUuid">
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.name }}{{ t.default ? '（默认）' : '' }}
          </option>
        </select>
      </div>
    </template>

    <h1>生成配置 URL</h1>

    <div class="section">
      <h2>Platform</h2>
      <select v-model="platform">
        <option value="windows">Windows</option>
        <option value="linux">Linux</option>
        <option value="android">Android</option>
      </select>
    </div>

    <div class="section">
      <h2>Tun 模式</h2>
      <label class="check-row"><input v-model="tun" type="checkbox" /> 启用 Tun</label>
    </div>

    <div class="section">
      <h2>Mixed 模式</h2>
      <label class="check-row"><input v-model="mixed" type="checkbox" /> 启用 Mixed</label>
      <div v-if="mixed" class="sub-fields">
        <div class="field">
          <div class="field-title">Mixed 端口</div>
          <input v-model="mixedPort" type="number" placeholder="7899" />
        </div>
        <label class="check-row"><input v-model="sysProxy" type="checkbox" /> 系统代理</label>
        <label class="check-row"><input v-model="share" type="checkbox" /> 局域网共享</label>
      </div>
    </div>

    <div class="section">
      <h2>Web UI（clash_api，默认禁用）</h2>
      <label class="check-row"><input v-model="webui" type="checkbox" /> 启用 Web UI</label>
      <div v-if="webui" class="sub-fields">
        <div class="field">
          <div class="field-title">端口</div>
          <input v-model="webuiPort" type="number" placeholder="9090" />
        </div>
        <div class="field">
          <div class="field-title">Secret（可选）</div>
          <input v-model="webuiSecret" type="text" placeholder="留空则不设置" />
        </div>
      </div>
    </div>

    <button id="copyBtn" @click="copyUrl">复制 URL</button>
    <button id="openBtn" @click="openUrl">打开</button>
    <p v-if="error" class="error">{{ error }}</p>
    <input id="url-box" type="text" readonly :value="url || '请选择 provider'" />
  </PageShell>
</template>

<style scoped>
h1 {
  font-size: 1.2rem;
  margin-bottom: 1rem;
  color: var(--text-1);
}
.field {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.8rem;
}
label {
  font-size: 0.85rem;
  margin-bottom: 4px;
  color: var(--text-2);
}
.group-title {
  margin-top: 1rem;
}
input[type='text'],
input[type='number'],
select {
  padding: 0.45rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  font-size: 0.9rem;
  background: var(--surface-2);
  color: var(--text-1);
}
.import-btn {
  margin-top: 6px;
}
button {
  padding: 0.5rem 0.9rem;
  font-size: 0.9rem;
  border: none;
  border-radius: var(--radius-2);
  cursor: pointer;
  margin: 4px 8px 4px 0;
  transition: filter var(--ease-3) 0.15s;
}
button:hover {
  filter: brightness(0.92);
}
.section {
  background: var(--surface-2);
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  padding: 1rem;
  margin-bottom: 1rem;
  box-shadow: var(--shadow-2);
}
.section h2 {
  font-size: 0.9rem;
  margin-bottom: 0.8rem;
  color: var(--text-2);
}
.field-title {
  font-weight: var(--font-weight-6);
  margin-bottom: 6px;
  font-size: 0.9rem;
  color: var(--text-1);
}
.check-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  color: var(--text-1);
  cursor: pointer;
}
.check-row input[type='checkbox'] {
  width: 17px;
  height: 17px;
  accent-color: var(--brand);
}
.sub-fields {
  border-left: 3px solid var(--surface-3);
  padding-left: 12px;
  margin-top: 4px;
}
.sub-fields .field {
  margin-bottom: 0.6rem;
}
#copyBtn {
  background: var(--green-7);
  color: #fff;
}
#openBtn {
  background: var(--brand);
  color: #fff;
}
#url-box {
  width: 100%;
  margin-top: 10px;
  padding: 0.6rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  font-family: var(--font-monospace-code);
  font-size: 0.85rem;
  background: var(--surface-2);
  color: var(--text-1);
  word-break: break-all;
}
.error {
  color: var(--red-7);
  margin-top: 6px;
  font-size: 0.9rem;
}
</style>
