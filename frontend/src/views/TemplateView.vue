<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../composables/useApi'
import { useMonaco } from '../composables/useMonaco'
import type { TemplateInfo } from '../types'

const { editorEl, setValue, getValue, format } = useMonaco()

const templates = ref<TemplateInfo[]>([])
const currentUuid = ref<string | null>(null)
const currentName = ref('')
const error = ref('')
const notice = ref('')
// 模板约定提示框：默认收起，避免占编辑空间
const hintOpen = ref(false)

async function loadList(): Promise<void> {
  try {
    templates.value = await api<TemplateInfo[]>('/api/templates')
  } catch (e) {
    error.value = '加载模板列表失败: ' + (e as Error).message
  }
}

async function openTemplate(uuid: string): Promise<void> {
  try {
    const data = await api<unknown>(`/api/templates/${uuid}`)
    const info = templates.value.find((t) => t.uuid === uuid)
    currentUuid.value = uuid
    currentName.value = (info ? info.name : uuid) + (info?.default ? '（默认）' : '')
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
    await openTemplate(created.uuid)
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
    notice.value = '已保存'
  } catch (e) {
    error.value = '保存失败: ' + (e as Error).message
  }
}

function formatTemplate(): void {
  try {
    JSON.parse(getValue())
    format()
    notice.value = '已格式化'
  } catch (e) {
    error.value = '格式化失败（JSON 不合法）: ' + (e as Error).message
  }
}

async function setDefault(uuid: string): Promise<void> {
  try {
    await api(`/api/templates/${uuid}/default`, { method: 'POST' })
    await loadList()
  } catch (e) {
    error.value = '设置失败: ' + (e as Error).message
  }
}

async function removeTemplate(t: TemplateInfo): Promise<void> {
  if (t.default) {
    error.value = '默认模板不可删除'
    return
  }
  if (!confirm(`删除模板 ${t.name}？`)) return
  try {
    await api(`/api/templates/${t.uuid}`, { method: 'DELETE' })
    if (currentUuid.value === t.uuid) {
      currentUuid.value = null
      currentName.value = ''
      setValue('')
    }
    await loadList()
  } catch (e) {
    error.value = '删除失败: ' + (e as Error).message
  }
}

onMounted(loadList)
</script>

<template>
  <main>
    <aside>
      <h2>模板列表</h2>
      <button id="new-tpl" @click="createTemplate">+ 新建模板</button>
      <div
        v-for="t in templates"
        :key="t.uuid"
        class="tpl-item"
        :class="{ active: t.uuid === currentUuid }"
        @click="openTemplate(t.uuid)"
      >
        <span>{{ t.default ? '★ ' : '' }}{{ t.name }}</span>
        <span>
          <button v-if="!t.default" class="op" @click.stop="setDefault(t.uuid)">设默认</button>
          <button class="del" :title="t.default ? '默认模板不可删除' : '删除'" @click.stop="removeTemplate(t)">
            ✕
          </button>
        </span>
      </div>
    </aside>

    <div id="editor-wrap">
      <div class="toolbar">
        <span class="cur-name">{{ currentName }}</span>
        <button id="save" class="btn-save" @click="saveTemplate">保存</button>
        <button id="format" @click="formatTemplate">格式化</button>
        <span class="ok" v-if="notice">{{ notice }}</span>
        <span class="error" v-else-if="error">{{ error }}</span>
      </div>
      <div class="tpl-hint">
        <div class="hint-header" @click="hintOpen = !hintOpen">
          <span>模板约定</span>
          <span>{{ hintOpen ? '收起 ▲' : '展开 ▼' }}</span>
        </div>
        <div v-show="hintOpen" class="hint-body">
          <ol>
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
        </div>
      </div>
      <div ref="editorEl" id="editor"></div>
    </div>
  </main>
</template>

<style scoped>
main {
  flex: 1;
  display: flex;
  overflow: hidden;
}
aside {
  width: 260px;
  border-right: 1px solid var(--surface-3);
  background: var(--surface-2);
  padding: 1rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
aside h2 {
  font-size: 0.9rem;
  margin: 0 0 0.5rem;
  color: var(--text-2);
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
  padding: 0.4rem 0.8rem;
  background: var(--brand);
  color: #fff;
}
.tpl-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.4rem 0.6rem;
  border-radius: var(--radius-2);
  cursor: pointer;
  color: var(--text-1);
}
.tpl-item:hover {
  background: var(--surface-3);
}
.tpl-item.active {
  background: var(--indigo-2);
  color: var(--indigo-10);
}
.tpl-item .del {
  color: var(--red-7);
  background: none;
  font-size: 0.9rem;
}
.tpl-item .op {
  color: var(--indigo-8);
  background: none;
  font-size: 0.8rem;
  margin-right: 0.3rem;
}
#editor-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 1rem;
}
.toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.5rem;
}
.toolbar .cur-name {
  font-size: 0.9rem;
  color: var(--text-1);
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
#editor {
  flex: 1;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  overflow: hidden;
}
.tpl-hint {
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  background: var(--surface-2);
  margin-bottom: 0.6rem;
  font-size: 0.85rem;
  flex-shrink: 0;
  max-width: 100%;
}
.hint-header {
  display: flex;
  justify-content: space-between;
  padding: 0.4rem 0.8rem;
  cursor: pointer;
  color: var(--text-2);
  font-weight: var(--font-weight-6);
  user-select: none;
}
.hint-body {
  padding: 0 0.8rem 0.6rem;
  color: var(--text-1);
  border-top: 1px solid var(--surface-3);
}
.hint-body ol {
  margin: 0.4rem 0 0;
  padding-left: 1.1rem;
}
.hint-body li {
  margin: 0.35rem 0;
}
.hint-body ul {
  margin: 0.2rem 0;
  padding-left: 1.2rem;
}
.hint-body code {
  background: var(--surface-3);
  padding: 0 4px;
  border-radius: 4px;
  font-family: var(--font-monospace-code);
  font-size: 0.8rem;
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
