<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PageShell from '../components/PageShell.vue'
import { api } from '../composables/useApi'
import type { ProviderConfig, ProviderRequest } from '../types'

const providers = ref<ProviderConfig[]>([])
const error = ref('')
const notice = ref('')

// 表单状态
const formTitle = ref('添加 Provider')
const submitLabel = ref('添加')
const editingUuid = ref<string | null>(null)
const form = ref<ProviderRequest>({ name: '', url: '', source: 'url', message: '' })
const fileInput = ref<File | null>(null)

async function load(): Promise<void> {
  try {
    providers.value = await api<ProviderConfig[]>('/api/providers')
    error.value = ''
  } catch (e) {
    error.value = '加载列表失败: ' + (e as Error).message
  }
}

function resetForm(): void {
  editingUuid.value = null
  form.value = { name: '', url: '', source: 'url', message: '' }
  fileInput.value = null
  formTitle.value = '添加 Provider'
  submitLabel.value = '添加'
}

function fillForm(p: ProviderConfig): void {
  editingUuid.value = p.uuid
  form.value = { name: p.name, url: p.url || '', source: p.source, message: p.message || '' }
  fileInput.value = null
  formTitle.value = '编辑 Provider'
  submitLabel.value = '保存修改'
}

async function uploadFile(uuid: string): Promise<void> {
  if (!fileInput.value) return
  const fd = new FormData()
  fd.append('file', fileInput.value)
  await api(`/api/providers/${uuid}/upload`, { method: 'POST', body: fd })
}

async function submit(): Promise<void> {
  if (form.value.source === 'url' && !/^https?:\/\//i.test(form.value.url)) {
    error.value = 'URL 必须是 http(s) 地址'
    return
  }
  try {
    if (editingUuid.value) {
      await api(`/api/providers/${editingUuid.value}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form.value),
      })
      await uploadFile(editingUuid.value)
    } else {
      const created = await api<ProviderConfig>('/api/providers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form.value),
      })
      await uploadFile(created.uuid)
    }
    notice.value = editingUuid.value ? '已保存' : '已添加'
    resetForm()
    await load()
  } catch (e) {
    error.value = '保存失败: ' + (e as Error).message
  }
}

async function remove(p: ProviderConfig): Promise<void> {
  if (!confirm(`删除 provider ${p.name}？`)) return
  try {
    await api(`/api/providers/${p.uuid}`, { method: 'DELETE' })
    await load()
  } catch (e) {
    error.value = '删除失败: ' + (e as Error).message
  }
}

async function fetchSub(p: ProviderConfig): Promise<void> {
  if (!confirm(`更新 ${p.name} 的订阅？`)) return
  try {
    const data = await api<{ bytes: number }>(`/api/providers/${p.uuid}/fetch`, { method: 'POST' })
    notice.value = `更新成功: ${data.bytes} bytes`
  } catch (e) {
    error.value = '更新失败: ' + (e as Error).message
  }
}

function uploadSub(p: ProviderConfig): void {
  const input = document.createElement('input')
  input.type = 'file'
  input.onchange = async () => {
    const f = input.files?.[0]
    if (!f) return
    const fd = new FormData()
    fd.append('file', f)
    try {
      await api(`/api/providers/${p.uuid}/upload`, { method: 'POST', body: fd })
      notice.value = '上传成功'
      await load()
    } catch (e) {
      error.value = '上传失败: ' + (e as Error).message
    }
  }
  input.click()
}

onMounted(load)
</script>

<template>
  <PageShell>
    <template #aside>
      <h2>{{ formTitle }}</h2>
      <form @submit.prevent="submit">
        <div class="field">
          <label>名称</label>
          <input v-model="form.name" placeholder="机场名称" required />
        </div>
        <div class="field">
          <label>来源</label>
          <select v-model="form.source">
            <option value="url">url</option>
            <option value="upload">upload</option>
          </select>
        </div>
        <div v-if="form.source === 'url'" class="field">
          <label>订阅 URL（http/https）</label>
          <input v-model="form.url" placeholder="https://..." />
        </div>
        <div v-else class="field">
          <label>上传配置文件</label>
          <input type="file" @change="fileInput = ($event.target as HTMLInputElement).files?.[0] ?? null" />
        </div>
        <div class="field">
          <label>备注</label>
          <input v-model="form.message" placeholder="可选" />
        </div>
        <button type="submit">{{ submitLabel }}</button>
      </form>
    </template>

    <h1>Provider 管理</h1>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="notice" class="ok">{{ notice }}</p>
    <table>
      <thead>
        <tr>
          <th>名称</th>
          <th>URL / 文件</th>
          <th>来源</th>
          <th>备注</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in providers" :key="p.uuid">
          <td class="name">{{ p.name }}</td>
          <td class="msg ellipsis" :title="p.source === 'upload' ? (p.file_name || '') : p.url">
            {{ p.source === 'upload' ? p.file_name || '' : p.url }}
          </td>
          <td class="src">{{ p.source }}</td>
          <td class="msg" :title="p.message || ''">{{ p.message || '' }}</td>
          <td class="actions">
            <button @click="fillForm(p)">编辑</button>
            <button class="btn-fetch" @click="p.source === 'upload' ? uploadSub(p) : fetchSub(p)">
              {{ p.source === 'upload' ? '上传' : '更新' }}
            </button>
            <button class="btn-del" @click="remove(p)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
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
input,
select {
  padding: 0.45rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  font-size: 0.9rem;
  background: var(--surface-2);
  color: var(--text-1);
}
button {
  padding: 0.5rem 0.9rem;
  border: none;
  border-radius: var(--radius-2);
  background: var(--brand);
  color: #fff;
  cursor: pointer;
  font-size: 0.9rem;
  transition: filter var(--ease-3) 0.15s;
}
button:hover {
  filter: brightness(0.92);
}
table {
  border-collapse: collapse;
  width: 100%;
  table-layout: fixed;
  background: var(--surface-2);
}
th,
td {
  border-bottom: 1px solid var(--surface-3);
  padding: 0.5rem;
  text-align: left;
  font-size: 0.9rem;
}
th {
  background: var(--surface-3);
  color: var(--text-1);
}
th:first-child {
  border-top-left-radius: var(--radius-2);
}
th:last-child {
  border-top-right-radius: var(--radius-2);
}
tr:last-child td {
  border-bottom: none;
}
tr:last-child td:first-child {
  border-bottom-left-radius: var(--radius-2);
}
tr:last-child td:last-child {
  border-bottom-right-radius: var(--radius-2);
}
.msg {
  color: var(--text-2);
  font-size: 0.82rem;
}
/* 长 URL/备注单行省略，悬停显示完整内容 */
.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src {
  width: 76px;
}
.actions {
  width: 150px;
  white-space: nowrap;
}
.error {
  color: var(--red-7);
  margin: 0.5rem 0;
  white-space: pre-wrap;
  font-size: 0.9rem;
}
.ok {
  color: var(--green-7);
  margin: 0.5rem 0;
  font-size: 0.9rem;
}
.actions button {
  margin-right: 0.3rem;
  padding: 0.3rem 0.7rem;
  font-size: 0.85rem;
}
.actions .btn-fetch {
  background: var(--green-7);
}
.actions .btn-del {
  background: var(--red-7);
}
</style>
