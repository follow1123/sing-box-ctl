<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import ContentPage from '../components/ContentPage.vue'
import Modal from '../components/Modal.vue'
import VersionDialog, { type VersionItem } from '../components/VersionDialog.vue'
import { api } from '../composables/useApi'
import type { ProviderConfig, ProviderRequest } from '../types'

const providers = ref<ProviderConfig[]>([])
const error = ref('')
const notice = ref('')

// 表单弹框状态
const showForm = ref(false)
const formTitle = ref('添加 Provider')
const submitLabel = ref('添加')
const editingUuid = ref<string | null>(null)
const form = ref<ProviderRequest>({ name: '', url: '', source: 'url', message: '' })
const fileInput = ref<File | null>(null)

// 历史版本弹框
const showVersions = ref(false)
const versionItems = ref<VersionItem[]>([])
const versionUuid = ref('')
const VERSION_LABELS: Record<string, string> = {
  current: '当前版本',
  last: '上次版本',
  old: '上上次版本',
}

async function openVersions(p: ProviderConfig): Promise<void> {
  try {
    const vers = await api<string[]>(`/api/providers/${p.uuid}/versions`)
    versionItems.value = vers.map((v) => ({
      version: v,
      label: VERSION_LABELS[v] ?? v,
      isCurrent: v === 'current',
    }))
    versionUuid.value = p.uuid
    showVersions.value = true
    error.value = ''
  } catch (e) {
    error.value = '加载版本失败: ' + (e as Error).message
  }
}

async function restoreVersion(version: string): Promise<void> {
  try {
    await api(`/api/providers/${versionUuid.value}/restore?version=${encodeURIComponent(version)}`, {
      method: 'POST',
    })
    showVersions.value = false
    notice.value = '已还原为' + (VERSION_LABELS[version] ?? version)
    await load()
  } catch (e) {
    error.value = '还原失败: ' + (e as Error).message
  }
}

async function load(): Promise<void> {
  try {
    providers.value = await api<ProviderConfig[]>('/api/providers')
    error.value = ''
  } catch (e) {
    error.value = '加载列表失败: ' + (e as Error).message
  }
}

function openCreate(): void {
  error.value = ''
  editingUuid.value = null
  form.value = { name: '', url: '', source: 'url', message: '' }
  fileInput.value = null
  formTitle.value = '添加 Provider'
  submitLabel.value = '添加'
  showForm.value = true
}

function openEdit(p: ProviderConfig): void {
  error.value = ''
  editingUuid.value = p.uuid
  form.value = { name: p.name, url: p.url || '', source: p.source, message: p.message || '' }
  fileInput.value = null
  formTitle.value = '编辑 Provider'
  submitLabel.value = '保存修改'
  showForm.value = true
}

function closeForm(): void {
  showForm.value = false
}


function onFileChange(e: Event): void {
  const el = e.target as HTMLInputElement
  fileInput.value = el.files?.[0] ?? null
}

// 切换为 url 来源时清掉已选文件，避免提交时误传
watch(
  () => form.value.source,
  (src) => {
    if (src === 'url') fileInput.value = null
  },
)

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
  if (form.value.source === 'upload' && !fileInput.value) {
    error.value = '请选择要上传的配置文件'
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
    closeForm()
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
  <ContentPage>
    <div class="page-header">
      <h1>Provider 管理</h1>
      <button class="btn-add" @click="openCreate">+ 添加</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="notice" class="ok">{{ notice }}</p>

    <!-- 统一单列卡片列表（所有端同一套，无详情页，操作全在卡片上） -->
    <p v-if="providers.length === 0" class="empty-tip">暂无 Provider，点右上角“+ 添加”创建。</p>
    <div v-else class="pcard-list">
      <div v-for="p in providers" :key="p.uuid" class="pcard">
        <div class="pcard-head">
          <span class="pcard-name">{{ p.name }}</span>
          <span class="src-tag">{{ p.source }}</span>
        </div>
        <div class="pcard-src" :title="p.source === 'upload' ? (p.file_name || '') : p.url">
          {{ p.source === 'upload' ? (p.file_name || '已上传配置') : p.url }}
        </div>
        <div v-if="p.message" class="pcard-msg">{{ p.message }}</div>
        <div class="pcard-actions">
          <button class="btn-edit" @click="openEdit(p)">编辑</button>
          <button class="btn-fetch" @click="p.source === 'upload' ? uploadSub(p) : fetchSub(p)">
            {{ p.source === 'upload' ? '上传' : '更新' }}
          </button>
          <button class="btn-ver" @click="openVersions(p)">版本</button>
          <button class="btn-del" @click="remove(p)">删除</button>
        </div>
      </div>
    </div>

    <Modal v-if="showForm" :title="formTitle" @close="closeForm">
      <form @submit.prevent="submit">
        <p v-if="error" class="error">{{ error }}</p>
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
          <input type="file" @change="onFileChange" />
        </div>
        <div class="field">
          <label>备注</label>
          <input v-model="form.message" placeholder="可选" />
        </div>
        <div class="form-actions">
          <button type="submit" class="btn-save">{{ submitLabel }}</button>
          <button type="button" class="btn-cancel" @click="closeForm">取消</button>
        </div>
      </form>
    </Modal>

    <!-- 历史版本弹框 -->
    <VersionDialog
      v-if="showVersions"
      :items="versionItems"
      empty="暂无版本记录，先更新或上传订阅一次"
      @close="showVersions = false"
      @restore="restoreVersion"
    />
  </ContentPage>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
h1 {
  font-size: 1.2rem;
  margin: 0;
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
.form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}
.form-actions .btn-save {
  background: var(--green-7);
}
.form-actions .btn-cancel {
  background: var(--surface-3);
  color: var(--text-1);
}
.btn-add {
  background: var(--brand);
  color: #fff;
}
/* 统一单列卡片列表（所有断点） */
.pcard-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.pcard {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
  padding: 0.7rem 0.85rem;
  background: var(--surface-2);
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
}
.pcard-head {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}
.pcard-name {
  font-size: 0.95rem;
  font-weight: var(--font-weight-6);
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-tag {
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: 0.7rem;
  background: var(--surface-3);
  border-radius: 999px;
  color: var(--text-2);
}
.pcard-src {
  font-size: 0.78rem;
  color: var(--text-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pcard-msg {
  font-size: 0.75rem;
  color: var(--text-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 操作：小按钮、统一靠左，分隔线隔开 */
.pcard-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.45rem;
  padding-top: 0.45rem;
  border-top: 1px solid var(--surface-3);
}
.pcard-actions button {
  padding: 0.28rem 0.7rem;
  font-size: 0.8rem;
}
.empty-tip {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-2);
}
.btn-detail,
.btn-del {
  color: #fff;
}
.btn-del {
  background: var(--red-7);
}
.btn-fetch {
  background: var(--green-7);
  color: #fff;
}
.btn-ver {
  background: var(--indigo-7);
  color: #fff;
}
.btn-edit {
  background: var(--brand);
  color: #fff;
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
</style>
