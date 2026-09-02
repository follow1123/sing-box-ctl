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
// Android 必须使用 Tun：此时 Tun 开关被强制开启且不可取消
const tunDisabled = computed(() => platform.value === 'android')
const tun = ref(false)
const mixed = ref(false)
const mixedListen = ref('')
const mixedPort = ref('')
const sysProxy = ref(false)
// sing-box api service（services.api，gRPC）
const apiEnabled = ref(false)
const apiListen = ref('')
const apiPort = ref('')
const apiSecret = ref('')
// dashboard 为 api service 的子配置，默认开启，仅在关闭时输出 api-dashboard=false
const apiDashboard = ref(true)
// 模板是否含常驻 inbound（mixed/tun 之外，无需开关也能生效）
const hasOtherInbound = ref(false)
// 模板是否含 tun inbound（Android 平台必需）
const hasTun = ref(false)
// 模板默认值（勾选 mixed/api 时才应用到输入框，避免模板全量铺开）
const tplDefaults = ref({
  mixed: { listen: '', port: '' },
  api: { listen: '', port: '', secret: '', dashboard: true },
})
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
    if (mixedListen.value) params.push('mixed-listen=' + mixedListen.value)
    if (mixedPort.value) params.push('mixed-port=' + mixedPort.value)
    if (sysProxy.value) params.push('sys-proxy')
  }
  if (apiEnabled.value) {
    params.push('api')
    if (!apiDashboard.value) params.push('api-dashboard=false')
    if (apiListen.value) params.push('api-listen=' + apiListen.value)
    if (apiPort.value) params.push('api-port=' + apiPort.value)
    if (apiSecret.value) params.push('api-secret=' + encodeURIComponent(apiSecret.value))
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
    mixedListen.value = q.get('mixed-listen') || ''
    mixedPort.value = q.get('mixed-port') || ''
    sysProxy.value = q.has('sys-proxy')
    apiEnabled.value = q.has('api')
    apiDashboard.value = q.get('api-dashboard') !== 'false'
    apiListen.value = q.get('api-listen') || ''
    apiPort.value = q.get('api-port') || ''
    apiSecret.value = q.get('api-secret') || ''
    syncPlatformState()
  } catch (e) {
    error.value = '导入失败: ' + (e as Error).message
  }
}

/** 平台状态同步：Android 强制启用 Tun */
function syncPlatformState(): void {
  if (platform.value === 'android') tun.value = true
}

function onPlatformChange(): void {
  syncPlatformState()
}

/** 勾选 mixed 时用模板默认值填充输入框 */
function fillMixedFromTemplate(): void {
  mixedListen.value = tplDefaults.value.mixed.listen
  mixedPort.value = tplDefaults.value.mixed.port
}

/** 勾选 api 时用模板默认值填充输入框 */
function fillApiFromTemplate(): void {
  apiListen.value = tplDefaults.value.api.listen
  apiPort.value = tplDefaults.value.api.port
  apiSecret.value = tplDefaults.value.api.secret
  apiDashboard.value = tplDefaults.value.api.dashboard
}

function onMixedChange(): void {
  if (mixed.value) fillMixedFromTemplate()
}

function onApiChange(): void {
  if (apiEnabled.value) fillApiFromTemplate()
}

/** 从模板读取默认配置并回填页面（模板内的 mixed/tun/api inbound 默认值） */
async function applyTemplateDefaults(): Promise<void> {
  if (!templateUuid.value) return
  try {
    const data = (await api<Record<string, any>>('/api/templates/' + templateUuid.value)) ?? {}
    const inbounds: any[] = data.inbounds ?? []
    // 默认模式 = 模板 inbounds 第一个的类型
    const firstInbound = inbounds[0]
    const mixedIn = inbounds.find((i) => i.type === 'mixed')
    const tunIn = inbounds.find((i) => i.type === 'tun')
    const services: any[] = data.services ?? []
    const apiSvc = services.find((s) => s.type === 'api')

    // 模板按默认平台（windows）编写，切换模板时平台复位
    platform.value = 'windows'
    syncPlatformState()
    hasTun.value = !!tunIn
    // mixed/tun 之外视为常驻 inbound，存在时无需勾选也能生成有效配置
    hasOtherInbound.value = inbounds.some((i) => i.type !== 'mixed' && i.type !== 'tun')

    // 维护模板默认值对象（勾选时才应用到输入框）
    tplDefaults.value = {
      mixed: {
        listen: mixedIn?.listen ?? '',
        port: mixedIn?.listen_port != null ? String(mixedIn.listen_port) : '',
      },
      api: apiSvc
        ? {
            listen: apiSvc.listen ?? '',
            port: apiSvc.listen_port != null ? String(apiSvc.listen_port) : '',
            secret: apiSvc.secret ?? '',
            dashboard:
              apiSvc.dashboard === true ||
              apiSvc.dashboard === undefined ||
              (apiSvc.dashboard && typeof apiSvc.dashboard === 'object' && apiSvc.dashboard.enabled !== false),
          }
        : { listen: '', port: '', secret: '', dashboard: true },
    }

    // 重置控件：清空输入框，默认只勾选模板第一个 inbound 对应模式（api 默认不勾）
    mixed.value = false
    tun.value = false
    mixedListen.value = ''
    mixedPort.value = ''
    apiEnabled.value = false
    apiListen.value = ''
    apiPort.value = ''
    apiSecret.value = ''
    apiDashboard.value = true
    if (firstInbound?.type === 'mixed') mixed.value = true
    else if (firstInbound?.type === 'tun') tun.value = true
    // 为默认勾选的模式填入模板默认值
    if (mixed.value) fillMixedFromTemplate()
  } catch (e) {
    error.value = '加载模板默认配置失败: ' + (e as Error).message
  }
}

async function copyUrl(): Promise<void> {
  if (!providerUuid.value) {
    error.value = '请先选择 provider'
    return
  }
  if (!tun.value && !mixed.value && !hasOtherInbound.value) {
    error.value = '需要至少启用 Tun 或 Mixed 一种模式'
    return
  }
  if (platform.value === 'android' && !hasTun.value) {
    error.value = '当前模板没有 Tun inbound，Android 无法使用'
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
  if (!tun.value && !mixed.value && !hasOtherInbound.value) {
    error.value = '需要至少启用 Tun 或 Mixed 一种模式'
    return
  }
  if (platform.value === 'android' && !hasTun.value) {
    error.value = '当前模板没有 Tun inbound，Android 无法使用'
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
    // 初始回填默认模板配置
    await applyTemplateDefaults()
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
        <select v-model="templateUuid" @change="applyTemplateDefaults">
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.name }}{{ t.default ? '（默认）' : '' }}
          </option>
        </select>
      </div>
    </template>

    <h1>生成配置 URL</h1>

    <div class="section">
      <h2>Platform</h2>
      <select v-model="platform" @change="onPlatformChange">
        <option value="windows">Windows</option>
        <option value="linux">Linux</option>
        <option value="android">Android</option>
      </select>
      <p v-if="platform === 'android'" class="section-desc">Android 必须使用 Tun 模式，Tun 开关已强制开启。</p>
    </div>

    <div class="section">
      <h2>Tun 模式</h2>
      <label class="check-row"><input v-model="tun" type="checkbox" :disabled="tunDisabled" /> 启用 Tun</label>
    </div>

    <div class="section">
      <h2>Mixed 模式</h2>
      <label class="check-row"><input v-model="mixed" type="checkbox" @change="onMixedChange" /> 启用 Mixed</label>
      <div v-if="mixed" class="sub-fields">
        <div class="field">
          <div class="field-title">监听地址</div>
          <input v-model="mixedListen" type="text" placeholder="127.0.0.1" />
        </div>
        <div class="field">
          <div class="field-title">端口</div>
          <input v-model="mixedPort" type="number" placeholder="7899" />
        </div>
        <label class="check-row"><input v-model="sysProxy" type="checkbox" /> 系统代理</label>
      </div>
    </div>

    <div class="section">
      <h2>Sing-box API</h2>
      <p class="section-desc">gRPC 服务（默认禁用），供 sing-box 官方客户端与 Dashboard 远程查看和控制本实例。</p>
      <label class="check-row"><input v-model="apiEnabled" type="checkbox" @change="onApiChange" /> 启用 Sing-box API</label>
      <div v-if="apiEnabled" class="sub-fields">
        <label class="check-row"><input v-model="apiDashboard" type="checkbox" /> 启用 Dashboard（Web 面板）</label>
        <div class="field">
          <div class="field-title">监听地址</div>
          <input v-model="apiListen" type="text" placeholder="127.0.0.1" />
        </div>
        <div class="field">
          <div class="field-title">端口</div>
          <input v-model="apiPort" type="number" placeholder="9090" />
        </div>
        <div class="field">
          <div class="field-title">Secret（可选）</div>
          <input v-model="apiSecret" type="text" placeholder="留空则不设置" />
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
.section-desc {
  font-size: 0.8rem;
  color: var(--text-2);
  margin: -0.4rem 0 0.6rem;
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
