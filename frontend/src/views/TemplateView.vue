<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Modal from '../components/Modal.vue'
import PinIcon from '../components/PinIcon.vue'
import VersionDialog, { type VersionItem } from '../components/VersionDialog.vue'
import { api } from '../composables/useApi'
import { useMonaco } from '../composables/useMonaco'
import type { TemplateInfo } from '../types'

const { editorEl, setValue, getValue, format } = useMonaco()

// 内嵌种子模板的特殊 key（与后端一致，不落盘）
const BUILTIN = 'builtin'

// 模板列表即 tab 集合：有几个模板就显示几个 tab
const templates = ref<TemplateInfo[]>([])
const currentUuid = ref<string | null>(null)
const showHelp = ref(false)
const showCreate = ref(false)
const createFrom = ref(BUILTIN)
const createName = ref('')
const createError = ref('')
// 未保存修改确认（切换 tab 前）
const showUnsaved = ref(false)
const pendingUuid = ref<string | null>(null)
// 历史版本弹框与还原确认
const showVersions = ref(false)
const versionItems = ref<VersionItem[]>([])
const showRestoreConfirm = ref(false)
const restoreVer = ref('')
const VERSION_LABELS: Record<string, string> = {
  current: '当前版本',
  last: '上次版本',
  old: '上上次版本',
}
// 当前编辑器内容对应的“已保存”快照，用于脏检测
const savedContent = ref('')
const error = ref('')
const notice = ref('')
let noticeTimer: ReturnType<typeof setTimeout> | undefined

function flashNotice(msg: string): void {
  error.value = ''
  notice.value = msg
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => (notice.value = ''), 2000)
}

function currentInfo(): TemplateInfo | undefined {
  return templates.value.find((t) => t.uuid === currentUuid.value)
}

async function loadList(): Promise<void> {
  try {
    templates.value = await api<TemplateInfo[]>('/api/templates')
  } catch (e) {
    error.value = '加载模板列表失败: ' + (e as Error).message
  }
}

/** 激活某模板：读取内容到编辑器（不做本地缓存） */
/** 当前是否有未保存的修改 */
function isDirty(): boolean {
  return currentUuid.value !== null && getValue() !== savedContent.value
}

/** 打开/切换到某模板（不做本地缓存）；有未保存修改时先弹确认 */
async function activate(uuid: string): Promise<void> {
  if (currentUuid.value === uuid && getValue() !== '') return
  if (currentUuid.value !== null && isDirty()) {
    pendingUuid.value = uuid
    showUnsaved.value = true
    return
  }
  await doActivate(uuid)
}

async function doActivate(uuid: string): Promise<void> {
  currentUuid.value = uuid
  try {
    const data = await api<unknown>(`/api/templates/${uuid}`)
    setValue(JSON.stringify(data, null, 2))
    savedContent.value = getValue()
    error.value = ''
  } catch (e) {
    error.value = '加载失败: ' + (e as Error).message
  }
}

/** 未保存确认弹框：先保存当前模板，成功后再切换 */
async function confirmSwitchWithSave(): Promise<void> {
  const ok = await saveTemplate()
  if (!ok) return
  const target = pendingUuid.value
  showUnsaved.value = false
  pendingUuid.value = null
  if (target) await doActivate(target)
}

/** 未保存确认弹框：取消切换，留在当前模板 */
function cancelSwitch(): void {
  showUnsaved.value = false
  pendingUuid.value = null
}

/** 打开历史版本弹框 */
async function openVersions(): Promise<void> {
  if (!currentUuid.value) return
  try {
    const vers = await api<string[]>(`/api/templates/${currentUuid.value}/versions`)
    versionItems.value = vers.map((v) => ({
      version: v,
      label: VERSION_LABELS[v] ?? v,
      isCurrent: v === 'current',
    }))
    showVersions.value = true
  } catch (e) {
    error.value = '加载版本失败: ' + (e as Error).message
  }
}

/** 点击某个历史版本的“还原”：有未保存修改时先弹三选确认 */
function onVersionRestore(version: string): void {
  if (isDirty()) {
    restoreVer.value = version
    showRestoreConfirm.value = true
    return
  }
  void doRestore(version, false)
}

function cancelRestoreConfirm(): void {
  showRestoreConfirm.value = false
  restoreVer.value = ''
}

/** 执行还原；saveFirst 为 true 时先保存当前编辑（保存后当前内容成为“上次版本”） */
async function doRestore(version: string, saveFirst: boolean): Promise<void> {
  if (!currentUuid.value) return
  if (saveFirst && !(await saveTemplate())) return // 保存失败则中止还原
  try {
    await api(`/api/templates/${currentUuid.value}/restore?version=${encodeURIComponent(version)}`, {
      method: 'POST',
    })
    showRestoreConfirm.value = false
    restoreVer.value = ''
    showVersions.value = false
    flashNotice('已还原为' + (VERSION_LABELS[version] ?? version))
    // 编辑器重载为还原后的内容
    await doActivate(currentUuid.value)
  } catch (e) {
    error.value = '还原失败: ' + (e as Error).message
  }
}

function confirmRestoreSave(): void {
  void doRestore(restoreVer.value, true)
}

function confirmRestoreDiscard(): void {
  void doRestore(restoreVer.value, false)
}

/** 打开新建弹框 */
function openCreate(): void {
  createError.value = ''
  createFrom.value = BUILTIN
  createName.value = ''
  showCreate.value = true
}

/** 新建模板：从内置种子或某个已有用户模板复制内容 */
async function submitCreate(): Promise<void> {
  const name = createName.value.trim()
  if (!name) {
    createError.value = '请输入模板名称'
    return
  }
  if (templates.value.some((t) => t.name === name)) {
    createError.value = `已存在同名模板“${name}”`
    return
  }
  try {
    const created = await api<TemplateInfo>(
      '/api/templates?name=' + encodeURIComponent(name) + '&from=' + encodeURIComponent(createFrom.value),
      { method: 'POST' },
    )
    await loadList()
    await activate(created.uuid)
    showCreate.value = false
    flashNotice('已创建')
  } catch (e) {
    createError.value = '新建失败: ' + (e as Error).message
  }
}

async function saveTemplate(): Promise<boolean> {
  if (!currentUuid.value) {
    error.value = '请先选择模板'
    return false
  }
  try {
    JSON.parse(getValue())
    await api(`/api/templates/${currentUuid.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: getValue(),
    })
    savedContent.value = getValue()
    flashNotice('已保存')
    return true
  } catch (e) {
    error.value = '保存失败: ' + (e as Error).message
    return false
  }
}

function formatTemplate(): void {
  try {
    JSON.parse(getValue())
    format()
    flashNotice('已格式化')
  } catch (e) {
    error.value = '格式化失败（JSON 不合法）: ' + (e as Error).message
  }
}

async function setCurrentDefault(): Promise<void> {
  if (!currentUuid.value) return
  try {
    await api(`/api/templates/${currentUuid.value}/default`, { method: 'POST' })
    await loadList()
    flashNotice('已设为默认')
  } catch (e) {
    error.value = '设置失败: ' + (e as Error).message
  }
}

async function removeCurrent(): Promise<void> {
  const t = currentInfo()
  if (!t || !currentUuid.value) return
  if (t.default) {
    error.value = '默认模板不可删除'
    return
  }
  if (!confirm(`删除模板 ${t.name}？`)) return
  try {
    await api(`/api/templates/${t.uuid}`, { method: 'DELETE' })
    // 当前模板已被删除，编辑内容随之作废，清空脏状态后自动切到默认/第一个模板
    showUnsaved.value = false
    pendingUuid.value = null
    await loadList()
    if (currentUuid.value === t.uuid) {
      currentUuid.value = null
      setValue('')
      savedContent.value = ''
    }
    const def = templates.value.find((x) => x.default) ?? templates.value[0]
    if (def) await doActivate(def.uuid)
  } catch (e) {
    error.value = '删除失败: ' + (e as Error).message
  }
}

onMounted(async () => {
  await loadList()
  const def = templates.value.find((t) => t.default) ?? templates.value[0]
  if (def) await activate(def.uuid)
})
</script>

<template>
  <main>
    <div class="page">
      <!-- 顶栏：固定工具按钮 + 可横向滚动的 tab 列表 -->
      <div class="tab-bar">
        <div class="tab-actions">
          <button class="tab-new" title="模板约定" @click="showHelp = true">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <circle cx="12" cy="12" r="10" />
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          </button>
          <button class="tab-new" title="新建模板" @click="openCreate">
            <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z" />
            </svg>
          </button>
        </div>
        <div class="tabs">
          <div
            v-for="t in templates"
            :key="t.uuid"
            class="tab"
            :class="{ active: t.uuid === currentUuid }"
            :title="t.default ? '默认模板' : ''"
            @click="activate(t.uuid)"
          >
            <span class="tab-name">
              <PinIcon v-if="t.default" class="pin-mark" />
              <span class="tab-text">{{ t.name }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="toolbar">
        <span class="cur-name">{{ currentInfo()?.name || '' }}</span>
        <button id="save" class="btn-save" @click="saveTemplate">保存</button>
        <button id="format" @click="formatTemplate">格式化</button>
        <template v-if="currentInfo()">
          <button id="versions" @click="openVersions">版本</button>
          <button v-if="!currentInfo()!.default" id="set-default" @click="setCurrentDefault">设为默认</button>
          <button v-if="!currentInfo()!.default" id="del-tpl" @click="removeCurrent">删除</button>
        </template>
        <span class="ok" v-if="notice">{{ notice }}</span>
        <span class="error" v-else-if="error">{{ error }}</span>
      </div>

      <div class="editor-area">
        <div ref="editorEl" id="editor"></div>
        <div v-if="templates.length === 0 && !currentUuid" class="empty-mask">
          <p class="empty-title">暂无模板</p>
          <p class="empty-tip">点击上方 “+ 新建”，从内置默认模板创建一个用户模板后再编辑。</p>
        </div>
      </div>
    </div>

    <!-- 新建模板 -->
    <Modal v-if="showCreate" title="新建模板" @close="showCreate = false">
      <p v-if="createError" class="error modal-error">{{ createError }}</p>
      <div class="field">
        <label>基于</label>
        <select v-model="createFrom">
          <option value="builtin">内置默认模板</option>
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.name }}{{ t.default ? '（默认）' : '' }}
          </option>
        </select>
      </div>
      <div class="field">
        <label>模板名称</label>
        <input v-model="createName" type="text" placeholder="输入新模板名称" @keyup.enter="submitCreate" />
      </div>
      <div class="form-actions">
        <button id="create-confirm" @click="submitCreate">创建</button>
        <button class="btn-cancel" @click="showCreate = false">取消</button>
      </div>
    </Modal>

    <!-- 未保存确认（切换模板前） -->
    <Modal v-if="showUnsaved" title="未保存的修改" @close="cancelSwitch">
      <p class="modal-tip">当前模板有未保存的修改，是否先保存再切换？</p>
      <div class="form-actions">
        <button id="save-switch" @click="confirmSwitchWithSave">保存</button>
        <button class="btn-cancel" @click="cancelSwitch">取消</button>
      </div>
    </Modal>

    <!-- 历史版本弹框 -->
    <VersionDialog
      v-if="showVersions"
      :items="versionItems"
      @close="showVersions = false"
      @restore="onVersionRestore"
    />

    <!-- 还原历史版本前，当前模板有未保存修改时的三选确认 -->
    <Modal v-if="showRestoreConfirm" title="还原历史版本" @close="cancelRestoreConfirm">
      <p class="modal-tip">当前模板有未保存的修改，请选择处理方式：</p>
      <p class="modal-sub">提示：选择“保存并还原”后，当前编辑会保存为“上次版本”，之后仍可还原回来。</p>
      <div class="form-actions">
        <button id="restore-save" @click="confirmRestoreSave">保存并还原</button>
        <button id="restore-discard" @click="confirmRestoreDiscard">丢弃并还原</button>
        <button class="btn-cancel" @click="cancelRestoreConfirm">取消</button>
      </div>
    </Modal>

    <!-- 模板约定帮助 -->
    <Modal v-if="showHelp" title="模板约定" @close="showHelp = false">
      <ol class="help-list">
        <li>URL 页默认勾选 <code>inbounds</code> 第一个类型作为默认模式，请把常用模式放第一个。</li>
        <li>建议同时配置 mixed 与 tun 两种 inbound（Android 强制 Tun）。</li>
        <li>
          outbound 的 tag 支持按关键词筛选订阅节点：
          <ul>
            <li><code>组名@all</code> 组包含全部节点</li>
            <li><code>组名@keywords=节点A,节点B</code> 仅包含名称命中任一关键词的节点</li>
            <li><code>组名@exclude=节点C</code> 排除名称命中任一关键词的节点</li>
            <li>不带 <code>@</code> 的组原样保留，需自行维护 outbounds</li>
          </ul>
        </li>
      </ol>
    </Modal>
  </main>
</template>

<style scoped>
main {
  flex: 1;
  display: flex;
  justify-content: center;
  overflow: hidden;
}
/* 内容居中（与其它页面一致），页面自身填满高度，编辑器不随页面滚动 */
.page {
  width: min(80%, 1200px);
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0.8rem 1rem 1rem;
  box-sizing: border-box;
}
@media (min-width: 768px) and (max-width: 1024px) {
  .page {
    width: 100%;
  }
}
@media (max-width: 767px) {
  .page {
    width: 100%;
    padding: 0.6rem 0.6rem 0.8rem;
  }
}
/* tab 栏容器：左侧固定工具按钮 + 右侧可滚动 tab 列表，共享底部边框 */
.tab-bar {
  display: flex;
  align-items: flex-end;
  gap: 0.25rem;
  border-bottom: 1px solid var(--surface-3);
  flex-shrink: 0;
}
.tab-actions {
  display: flex;
  align-items: flex-end;
  flex-shrink: 0;
}
button {
  border: none;
  border-radius: var(--radius-2);
  cursor: pointer;
  transition: filter var(--ease-3) 0.15s;
}
button:hover {
  filter: brightness(0.92);
}
.tabs {
  display: flex;
  align-items: flex-end;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
  min-width: 0;
  /* 隐藏横向滚动条（多 tab 时仍可横向滚动） */
  scrollbar-width: none;
}
.tabs::-webkit-scrollbar {
  display: none;
}
.tab-new {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 1.7rem;
  height: 1.7rem;
  margin: 0 0.2rem 0.15rem 0;
  padding: 0;
  background: none;
  color: var(--text-2);
  border-radius: var(--radius-1);
}
.tab-new:hover {
  background: var(--surface-3);
  color: var(--text-1);
}
.tab-new svg {
  width: 1rem;
  height: 1rem;
}
.tab {
  display: flex;
  align-items: center;
  padding: 0.4rem 0.8rem;
  font-size: 0.85rem;
  color: var(--text-2);
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: var(--radius-2) var(--radius-2) 0 0;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.tab:hover {
  background: var(--surface-3);
}
.tab.active {
  background: var(--surface-2);
  border-color: var(--surface-3);
  color: var(--text-1);
  position: relative;
  top: 1px;
}
.tab-name {
  display: inline-flex;
  align-items: center;
  max-width: 160px;
}
.tab-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pin-mark {
  margin-right: 0.25rem;
  flex-shrink: 0;
}
.toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.4rem 0;
  flex-shrink: 0;
}
@media (max-width: 767px) {
  .toolbar {
    flex-wrap: wrap;
    row-gap: 0.35rem;
  }
  .toolbar .cur-name {
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.toolbar .cur-name {
  font-size: 0.9rem;
  color: var(--text-1);
  font-weight: var(--font-weight-6);
  margin-right: auto;
}
.toolbar button {
  padding: 0.35rem 0.8rem;
  background: var(--surface-3);
  color: var(--text-1);
  font-size: 0.9rem;
}
.toolbar .btn-save {
  background: var(--green-7);
  color: #fff;
}
#set-default {
  background: var(--indigo-7);
  color: #fff;
}
#del-tpl {
  background: var(--red-7);
  color: #fff;
}
.editor-area {
  flex: 1;
  position: relative;
  min-height: 0;
}
#editor {
  position: absolute;
  inset: 0;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  overflow: hidden;
}
.empty-mask {
  position: absolute;
  inset: 0;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  background: var(--surface-2);
  border: 1px dashed var(--surface-4);
  border-radius: var(--radius-2);
}
.empty-title {
  font-size: 1rem;
  font-weight: var(--font-weight-7);
  color: var(--text-2);
  margin: 0;
}
.empty-tip {
  font-size: 0.85rem;
  color: var(--text-2);
  margin: 0;
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
.form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}
#create-confirm {
  background: var(--brand);
  color: #fff;
  padding: 0.5rem 1rem;
}
.form-actions .btn-cancel {
  background: var(--surface-3);
  color: var(--text-1);
  padding: 0.5rem 1rem;
}
.help-list {
  margin: 0;
  padding-left: 1.1rem;
}
.help-list li {
  margin: 0.4rem 0;
  font-size: 0.9rem;
}
.help-list ul {
  margin: 0.2rem 0;
  padding-left: 1.2rem;
}
.help-list code {
  background: var(--surface-3);
  padding: 0 4px;
  border-radius: 4px;
  font-family: var(--font-monospace-code);
  font-size: 0.85rem;
}
#save-switch {
  background: var(--green-7);
  color: #fff;
  padding: 0.5rem 1rem;
}
#restore-save {
  background: var(--green-7);
  color: #fff;
  padding: 0.5rem 1rem;
}
#restore-discard {
  background: var(--red-7);
  color: #fff;
  padding: 0.5rem 1rem;
}
.form-actions .btn-cancel {
  padding: 0.5rem 1rem;
}
.modal-sub {
  margin: 0 0 0.6rem;
  font-size: 0.85rem;
  color: var(--text-2);
}
.error {
  color: var(--red-7);
  margin-left: 0.5rem;
  white-space: pre-wrap;
  font-size: 0.85rem;
}
.modal-error {
  margin: 0 0 0.6rem;
  font-size: 0.85rem;
}
.modal-tip {
  margin: 0 0 0.4rem;
  font-size: 0.9rem;
  color: var(--text-1);
}
.ok {
  color: var(--green-7);
  margin-left: 0.5rem;
  font-size: 0.85rem;
}
</style>
