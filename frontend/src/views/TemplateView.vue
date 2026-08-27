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
