<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ContentPage from '../components/ContentPage.vue'
import { api } from '../composables/useApi'
import type { ProviderConfig, TemplateInfo } from '../types'

const providers = ref<ProviderConfig[]>([])
const templates = ref<TemplateInfo[]>([])
const error = ref('')
const notice = ref('')
let noticeTimer: ReturnType<typeof setTimeout> | undefined

function flashNotice(msg: string): void {
  error.value = ''
  notice.value = msg
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => (notice.value = ''), 2000)
}

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

/** 顶部“导入”按钮：弹窗输入 URL，取消/空输入不做任何处理 */
function promptImport(): void {
  const raw = window.prompt('粘贴已有的配置 URL')
  if (raw === null) return
  const text = raw.trim()
  if (!text) return
  importFromUrl(text)
}

/** 从配置 URL 解析并回填页面状态 */
function importFromUrl(raw: string): void {
  try {
    const u = new URL(raw, location.origin)
    const m = u.pathname.match(/\/config\/([^/]+)/)
    if (!m) {
      error.value = 'URL 格式不正确，应包含 /config/<provider-uuid>'
      return
    }
    providerUuid.value = m[1]
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
    notice.value = '已导入'
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
  if (templates.value.length === 0) {
    error.value = '暂无模板，请先在模板管理页新建模板'
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
    flashNotice('已复制到剪贴板')
  } catch {
    // 旧浏览器降级
    const ta = document.createElement('textarea')
    ta.value = url.value
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    if (ok) flashNotice('已复制到剪贴板')
    else error.value = '复制失败，请手动选择复制'
  }
}

function openUrl(): void {
  if (!providerUuid.value) {
    error.value = '请先选择 provider'
    return
  }
  if (templates.value.length === 0) {
    error.value = '暂无模板，请先在模板管理页新建模板'
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
  <ContentPage>
    <h1>生成配置 URL</h1>

    <!-- 区域1：按钮功能区 -->
    <div class="top-bar">
      <button class="btn-import" @click="promptImport">导入</button>
      <div class="top-actions">
        <button id="copyBtn" @click="copyUrl">复制 URL</button>
        <button id="openBtn" @click="openUrl">打开</button>
      </div>
    </div>

    <!-- 区域2：URL 预览 + 提示 -->
    <div class="card preview-card">
      <div class="card-title">URL 预览</div>
      <textarea
        id="url-box"
        readonly
        rows="3"
        :value="url || '请选择 Provider 以生成 URL'"
      ></textarea>
      <p v-if="error" class="error">{{ error }}</p>
      <p v-else-if="notice" class="ok">{{ notice }}</p>
    </div>

    <!-- 区域3：选择元数据 -->
    <h2 class="section-label">选择 Provider 与模板</h2>
    <div class="select-grid">
      <div class="card">
        <div class="card-title">Provider</div>
        <select v-model="providerUuid">
          <option value="">-- 选择 provider --</option>
          <option v-for="p in providers" :key="p.uuid" :value="p.uuid">{{ p.name }}</option>
        </select>
      </div>
      <div class="card">
        <div class="card-title">模板</div>
        <select v-model="templateUuid" @change="applyTemplateDefaults">
          <option v-if="templates.length === 0" value="" disabled>暂无模板，请先在模板页新建</option>
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.name }}{{ t.default ? '（默认）' : '' }}
          </option>
        </select>
      </div>
    </div>

    <!-- 区域4：配置开关 -->
    <h2 class="section-label">模式与功能开关</h2>
    <div class="option-grid">
      <div class="card">
        <div class="card-title">Platform</div>
        <select v-model="platform" @change="onPlatformChange">
          <option value="windows">Windows</option>
          <option value="linux">Linux</option>
          <option value="android">Android</option>
        </select>
        <p v-if="platform === 'android'" class="platform-hint">Android 必须使用 Tun 模式，Tun 开关已强制开启。</p>
      </div>

      <div class="card">
        <div class="card-title">Tun 模式</div>
        <label class="check-row"><input v-model="tun" type="checkbox" :disabled="tunDisabled" /> 启用 Tun</label>
      </div>

      <div class="card">
        <div class="card-title">Mixed 模式</div>
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

      <div class="card">
        <div class="card-title">Sing-box API</div>
        <p class="card-sub">gRPC 服务（默认禁用），供 sing-box 官方客户端与 Dashboard 远程查看和控制本实例。</p>
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
    </div>
  </ContentPage>
</template>

<style scoped>
h1 {
  font-size: 1.2rem;
  margin: 0 0 1rem;
  color: var(--text-1);
}
.top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 1rem;
}
.top-actions {
  display: flex;
  gap: 0.6rem;
}
button {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: var(--radius-2);
  background: var(--surface-3);
  color: var(--text-1);
  cursor: pointer;
  font-size: 0.9rem;
  transition: filter var(--ease-3) 0.15s;
}
button:hover {
  filter: brightness(0.92);
}
.btn-import {
  background: var(--surface-3);
}
#copyBtn {
  background: var(--green-7);
  color: #fff;
}
#openBtn {
  background: var(--brand);
  color: #fff;
}
.card {
  background: var(--surface-2);
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  padding: 1rem;
}
.card-title {
  font-weight: var(--font-weight-6);
  font-size: 0.95rem;
  color: var(--text-1);
  margin-bottom: 0.6rem;
}
/* 分区小标题：文字 + 右侧延伸分隔线 */
.section-label {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 1rem;
  font-weight: var(--font-weight-7);
  color: var(--text-1);
  margin: 1.8rem 0 1rem;
}
.section-label::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--surface-3);
}
.card-sub {
  font-size: 0.8rem;
  color: var(--text-2);
  margin: -0.2rem 0 0.6rem;
}
.preview-card {
  margin-bottom: 1rem;
}
#url-box {
  width: 100%;
  box-sizing: border-box;
  padding: 0.6rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  font-family: var(--font-monospace-code);
  font-size: 0.85rem;
  background: var(--surface-1);
  color: var(--text-1);
  resize: vertical;
  word-break: break-all;
  white-space: pre-wrap;
}
.select-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}
.option-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}
@media (max-width: 767px) {
  .select-grid,
  .option-grid {
    grid-template-columns: 1fr;
  }
}
select,
input {
  padding: 0.45rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  font-size: 0.9rem;
  background: var(--surface-1);
  color: var(--text-1);
  width: 100%;
  box-sizing: border-box;
}
.check-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  color: var(--text-1);
  cursor: pointer;
}
.check-row input[type='checkbox'] {
  width: auto;
}
.sub-fields {
  border-left: 3px solid var(--surface-4);
  padding-left: 12px;
  margin-top: 6px;
}
.field {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.6rem;
}
.field-title {
  font-weight: var(--font-weight-5);
  margin-bottom: 4px;
  font-size: 0.85rem;
  color: var(--text-2);
}
.platform-hint {
  font-size: 0.8rem;
  color: var(--orange-9);
  margin: 0.7rem 0 0;
}
.error {
  color: var(--red-7);
  margin: 0.6rem 0 0;
  font-size: 0.9rem;
  white-space: pre-wrap;
}
.ok {
  color: var(--green-7);
  margin: 0.6rem 0 0;
  font-size: 0.9rem;
}
</style>
