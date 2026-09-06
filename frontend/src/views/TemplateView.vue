<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Modal from '../components/Modal.vue'
import Btn from '../components/ui/Btn.vue'
import Field from '../components/ui/Field.vue'
import Select from '../components/ui/Select.vue'
import VersionDialog, { type VersionItem } from '../components/VersionDialog.vue'
import { api } from '../composables/useApi'
import { useMonaco } from '../composables/useMonaco'
import { toast } from '../composables/useToast'
import type { TemplateInfo } from '../types'

const { editorEl, setValue, getValue, format } = useMonaco()

// 内嵌种子模板的特殊 key（与后端一致，不落盘）
const BUILTIN = 'builtin'

const templates = ref<TemplateInfo[]>([])
const currentUuid = ref<string | null>(null)
// 下拉选择值（与 currentUuid 保持同步；切换被取消时回滚到 currentUuid）
const curSel = ref('')
const showHelp = ref(false)
const showCreate = ref(false)
const createFrom = ref(BUILTIN)
const createName = ref('')
// 未保存修改确认（切换模板前）
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

function currentInfo(): TemplateInfo | undefined {
  return templates.value.find((t) => t.uuid === currentUuid.value)
}

async function loadList(): Promise<void> {
  try {
    templates.value = await api<TemplateInfo[]>('/api/templates')
  } catch (e) {
    toast.error('加载模板列表失败: ' + (e as Error).message)
  }
}

/** 当前是否有未保存的修改 */
function isDirty(): boolean {
  return currentUuid.value !== null && getValue() !== savedContent.value
}

/** 下拉选择模板；有未保存修改时先弹确认 */
function onSelectChange(): void {
  void activate(curSel.value)
}

/** 打开/切换到某模板；有未保存修改时先弹确认 */
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
  curSel.value = uuid
  try {
    const data = await api<unknown>(`/api/templates/${uuid}`)
    setValue(JSON.stringify(data, null, 2))
    savedContent.value = getValue()
  } catch (e) {
    toast.error('加载失败: ' + (e as Error).message)
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
  curSel.value = currentUuid.value ?? ''
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
    toast.error('加载版本失败: ' + (e as Error).message)
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
    toast.info('已还原为' + (VERSION_LABELS[version] ?? version))
    // 编辑器重载为还原后的内容
    await doActivate(currentUuid.value)
  } catch (e) {
    toast.error('还原失败: ' + (e as Error).message)
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
  createFrom.value = BUILTIN
  createName.value = ''
  showCreate.value = true
}

/** 新建模板：从内置种子或某个已有用户模板复制内容 */
async function submitCreate(): Promise<void> {
  const name = createName.value.trim()
  if (!name) {
    toast.warn('请输入模板名称')
    return
  }
  if (templates.value.some((t) => t.name === name)) {
    toast.warn(`已存在同名模板“${name}”`)
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
    toast.info('已创建')
  } catch (e) {
    toast.error('新建失败: ' + (e as Error).message)
  }
}

async function saveTemplate(): Promise<boolean> {
  if (!currentUuid.value) {
    toast.warn('请先选择模板')
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
    toast.info('已保存')
    return true
  } catch (e) {
    toast.error('保存失败: ' + (e as Error).message)
    return false
  }
}

function formatTemplate(): void {
  try {
    JSON.parse(getValue())
    format()
    toast.info('已格式化')
  } catch {
    toast.error('格式化失败（JSON 不合法）')
  }
}

async function setCurrentDefault(): Promise<void> {
  if (!currentUuid.value) return
  try {
    await api(`/api/templates/${currentUuid.value}/default`, { method: 'POST' })
    await loadList()
    toast.info('已设为默认')
  } catch (e) {
    toast.error('设置失败: ' + (e as Error).message)
  }
}

async function removeCurrent(): Promise<void> {
  const t = currentInfo()
  if (!t || !currentUuid.value) return
  if (t.default) {
    toast.warn('默认模板不可删除')
    return
  }
  if (!confirm(`删除模板 ${t.name}？`)) return
  try {
    await api(`/api/templates/${t.uuid}`, { method: 'DELETE' })
    // 当前模板已被删除，编辑内容随之作废，自动切到默认/第一个模板
    showUnsaved.value = false
    pendingUuid.value = null
    await loadList()
    if (currentUuid.value === t.uuid) {
      currentUuid.value = null
      curSel.value = ''
      setValue('')
      savedContent.value = ''
    }
    const def = templates.value.find((x) => x.default) ?? templates.value[0]
    if (def) await doActivate(def.uuid)
    toast.info('已删除')
  } catch (e) {
    toast.error('删除失败: ' + (e as Error).message)
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
      <!-- 顶栏：左添加 + 中模板下拉 + 右操作按钮（移动端操作组换行） -->
      <div class="toolbar">
        <Btn variant="primary" @click="openCreate">+ 添加</Btn>

        <Select
          v-model="curSel"
          class="tpl-select"
          placeholder="暂无模板，请先添加"
          :disabled="templates.length === 0"
          @change="onSelectChange"
        >
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.default ? '默认 ' : '' }}{{ t.name }}
          </option>
        </Select>

        <div class="tpl-actions">
          <template v-if="currentInfo()">
            <Btn v-if="!currentInfo()!.default" @click="setCurrentDefault">设为默认</Btn>
            <Btn variant="primary" @click="saveTemplate">保存</Btn>
            <Btn @click="formatTemplate">格式化</Btn>
            <Btn variant="danger" :disabled="currentInfo()!.default" @click="removeCurrent">删除</Btn>
            <Btn @click="openVersions">版本</Btn>
          </template>
          <Btn @click="showHelp = true">说明</Btn>
        </div>
      </div>

      <div class="editor-area">
        <div ref="editorEl" id="editor"></div>
        <div v-if="templates.length === 0 && !currentUuid" class="empty-mask">
          <p class="empty-title">暂无模板</p>
          <p class="empty-tip">点击左侧 “+ 添加”，从内置默认模板创建一个用户模板后再编辑。</p>
        </div>
      </div>
    </div>

    <!-- 新建模板 -->
    <Modal v-if="showCreate" title="新建模板" @close="showCreate = false">
      <Field label="基于">
        <Select v-model="createFrom">
          <option value="builtin">内置默认模板</option>
          <option v-for="t in templates" :key="t.uuid" :value="t.uuid">
            {{ t.name }}{{ t.default ? '（默认）' : '' }}
          </option>
        </Select>
      </Field>
      <Field label="模板名称">
        <input
          v-model="createName"
          type="text"
          placeholder="输入新模板名称"
          @keyup.enter="submitCreate"
        />
      </Field>
      <div class="form-actions">
        <Btn variant="primary" @click="submitCreate">创建</Btn>
        <Btn @click="showCreate = false">取消</Btn>
      </div>
    </Modal>

    <!-- 未保存确认（切换模板前） -->
    <Modal v-if="showUnsaved" title="未保存的修改" @close="cancelSwitch">
      <p class="modal-tip">当前模板有未保存的修改，是否先保存再切换？</p>
      <div class="form-actions">
        <Btn variant="primary" @click="confirmSwitchWithSave">保存</Btn>
        <Btn @click="cancelSwitch">取消</Btn>
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
        <Btn variant="primary" @click="confirmRestoreSave">保存并还原</Btn>
        <Btn variant="danger" @click="confirmRestoreDiscard">丢弃并还原</Btn>
        <Btn @click="cancelRestoreConfirm">取消</Btn>
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
  height: 100%;
  display: flex;
  justify-content: center;
  overflow: hidden;
}
/* 页面占满骨架高度，编辑器不随页面滚动 */
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
/* 顶栏：添加 + 模板下拉 + 操作按钮（同行等尺寸；移动端操作组换行） */
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.1rem 0 0.55rem;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.tpl-select {
  flex: 1;
  min-width: 0;
}
.tpl-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
@media (max-width: 767px) {
  .tpl-actions {
    flex-basis: 100%;
    row-gap: 0.5rem;
  }
}
.editor-area {
  flex: 1;
  position: relative;
  min-height: 0;
}
#editor {
  position: absolute;
  inset: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
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
  background: var(--bg);
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-lg);
}
.empty-title {
  margin: 0;
  font-size: 1rem;
  font-weight: var(--font-weight-6);
  color: var(--text-secondary);
}
.empty-tip {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-faint);
}
.form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}
.modal-sub {
  margin: 0 0 0.6rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}
.modal-tip {
  margin: 0 0 0.4rem;
  font-size: 0.9rem;
  color: var(--text);
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
  padding: 0 4px;
  font-family: var(--font-mono);
  font-size: 0.85rem;
  background: var(--neutral-soft);
  border-radius: var(--radius-sm);
}
</style>
