<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Modal from '../components/Modal.vue'
import { api } from '../composables/useApi'
import { useMonaco } from '../composables/useMonaco'
import type { TemplateInfo } from '../types'

const { editorEl, setValue, getValue, format } = useMonaco()

// 模板列表即 tab 集合：有几个模板就显示几个 tab
const templates = ref<TemplateInfo[]>([])
const currentUuid = ref<string | null>(null)
const showHelp = ref(false)
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
async function activate(uuid: string): Promise<void> {
  if (currentUuid.value === uuid && getValue() !== '') return
  currentUuid.value = uuid
  try {
    const data = await api<unknown>(`/api/templates/${uuid}`)
    setValue(JSON.stringify(data, null, 2))
    error.value = ''
  } catch (e) {
    error.value = '加载失败: ' + (e as Error).message
  }
}

async function createTemplate(): Promise<void> {
  const name = prompt('输入模板名称')
  if (!name) return
  try {
    const created = await api<TemplateInfo>('/api/templates?name=' + encodeURIComponent(name), {
      method: 'POST',
    })
    await loadList()
    currentUuid.value = null // 强制重新读取新模板内容
    await activate(created.uuid)
  } catch (e) {
    error.value = '新建失败: ' + (e as Error).message
  }
}

async function saveTemplate(): Promise<void> {
  if (!currentUuid.value) {
    error.value = '请先选择模板'
    return
  }
  try {
    JSON.parse(getValue())
    await api(`/api/templates/${currentUuid.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: getValue(),
    })
    flashNotice('已保存')
  } catch (e) {
    error.value = '保存失败: ' + (e as Error).message
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
    await loadList()
    if (currentUuid.value === t.uuid) {
      currentUuid.value = null
      setValue('')
    }
    // 自动激活默认模板（或第一个），保持编辑器不空
    const def = templates.value.find((x) => x.default) ?? templates.value[0]
    if (def) await activate(def.uuid)
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
      <!-- 顶栏：新建按钮 + tab（= 全部模板）+ 约定帮助 -->
      <div class="top-row">
        <button id="new-tpl" @click="createTemplate">+ 新建</button>
        <div class="tabs">
          <div
            v-for="t in templates"
            :key="t.uuid"
            class="tab"
            :class="{ active: t.uuid === currentUuid }"
            :title="t.default ? '默认模板' : ''"
            @click="activate(t.uuid)"
          >
            <span class="tab-name">{{ t.default ? '★ ' : '' }}{{ t.name }}</span>
          </div>
        </div>
        <button class="help-btn" title="模板约定" @click="showHelp = true">?</button>
      </div>

      <div class="toolbar">
        <span class="cur-name">{{ currentInfo()?.name || '' }}</span>
        <button id="save" class="btn-save" @click="saveTemplate">保存</button>
        <button id="format" @click="formatTemplate">格式化</button>
        <template v-if="currentInfo()">
          <button v-if="!currentInfo()!.default" id="set-default" @click="setCurrentDefault">设为默认</button>
          <button v-if="!currentInfo()!.default" id="del-tpl" @click="removeCurrent">删除</button>
        </template>
        <span class="ok" v-if="notice">{{ notice }}</span>
        <span class="error" v-else-if="error">{{ error }}</span>
      </div>
      <div ref="editorEl" id="editor"></div>
    </div>

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
/* 内容居中 80%（与其它页面一致），页面自身填满高度，编辑器不随页面滚动 */
.page {
  width: min(80%, 1200px);
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0.8rem 1rem 1rem;
  box-sizing: border-box;
}
.top-row {
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
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
#new-tpl {
  padding: 0.45rem 0.8rem;
  background: var(--brand);
  color: #fff;
  font-size: 0.9rem;
  flex-shrink: 0;
}
.tabs {
  display: flex;
  overflow-x: auto;
  flex: 1;
  min-width: 0;
  border-bottom: 1px solid var(--surface-3);
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
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.help-btn {
  width: 1.6rem;
  height: 1.6rem;
  font-size: 0.9rem;
  background: var(--surface-3);
  color: var(--text-2);
  border-radius: 50%;
  flex-shrink: 0;
  margin-bottom: 0.4rem;
}
.help-btn:hover {
  background: var(--surface-4);
  color: var(--text-1);
}
.toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.4rem 0;
  flex-shrink: 0;
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
#editor {
  flex: 1;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  overflow: hidden;
  min-height: 0;
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
.error {
  color: var(--red-7);
  margin-left: 0.5rem;
  white-space: pre-wrap;
  font-size: 0.85rem;
}
.ok {
  color: var(--green-7);
  margin-left: 0.5rem;
  font-size: 0.85rem;
}
</style>
