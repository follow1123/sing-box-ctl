<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import ContentPage from '../components/ContentPage.vue'
import Modal from '../components/Modal.vue'
import Btn from '../components/ui/Btn.vue'
import Field from '../components/ui/Field.vue'
import VersionDialog, { type VersionItem } from '../components/VersionDialog.vue'
import { api } from '../composables/useApi'
import { toast } from '../composables/useToast'
import type { ProviderConfig, ProviderRequest } from '../types'

const providers = ref<ProviderConfig[]>([])

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
  } catch (e) {
    toast.error('加载版本失败: ' + (e as Error).message)
  }
}

async function restoreVersion(version: string): Promise<void> {
  try {
    await api(`/api/providers/${versionUuid.value}/restore?version=${encodeURIComponent(version)}`, {
      method: 'POST',
    })
    showVersions.value = false
    toast.info('已还原为' + (VERSION_LABELS[version] ?? version))
    await load()
  } catch (e) {
    toast.error('还原失败: ' + (e as Error).message)
  }
}

async function load(): Promise<void> {
  try {
    providers.value = await api<ProviderConfig[]>('/api/providers')
  } catch (e) {
    toast.error('加载失败: ' + (e as Error).message)
  }
}

function openCreate(): void {
  editingUuid.value = null
  form.value = { name: '', url: '', source: 'url', message: '' }
  fileInput.value = null
  formTitle.value = '添加 Provider'
  submitLabel.value = '添加'
  showForm.value = true
}

function openEdit(p: ProviderConfig): void {
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
    toast.warn('URL 必须是 http(s) 地址')
    return
  }
  // 新增 upload 来源必须带文件；编辑时可不换文件（只改元数据）
  if (!editingUuid.value && form.value.source === 'upload' && !fileInput.value) {
    toast.warn('请选择要上传的配置文件')
    return
  }
  try {
    if (editingUuid.value) {
      // 编辑：元数据走 JSON PUT；upload 来源若重新选了文件则再上传覆盖
      await api(`/api/providers/${editingUuid.value}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form.value),
      })
      if (form.value.source === 'upload' && fileInput.value) {
        await uploadFile(editingUuid.value)
      }
    } else {
      // 新增：multipart 一步式——后端解析出节点成功后才创建条目
      const fd = new FormData()
      fd.append('name', form.value.name)
      fd.append('source', form.value.source)
      if (form.value.message) fd.append('message', form.value.message)
      if (form.value.source === 'url') fd.append('url', form.value.url)
      if (form.value.source === 'upload' && fileInput.value) fd.append('file', fileInput.value)
      await api('/api/providers', { method: 'POST', body: fd })
    }
    toast.info((editingUuid.value ? '已保存' : '已添加') + '「' + form.value.name + '」')
    closeForm()
    await load()
  } catch (e) {
    toast.error('保存失败: ' + (e as Error).message)
  }
}

async function remove(p: ProviderConfig): Promise<void> {
  if (!confirm(`删除 provider ${p.name}？`)) return
  try {
    await api(`/api/providers/${p.uuid}`, { method: 'DELETE' })
    toast.info('已删除「' + p.name + '」')
    await load()
  } catch (e) {
    toast.error('删除失败: ' + (e as Error).message)
  }
}

async function setDefault(p: ProviderConfig): Promise<void> {
  try {
    await api(`/api/providers/${p.uuid}/default`, { method: 'POST' })
    toast.info('已设「' + p.name + '」为默认')
    await load()
  } catch (e) {
    toast.error('设置默认失败: ' + (e as Error).message)
  }
}

async function fetchSub(p: ProviderConfig): Promise<void> {
  if (!confirm(`更新 ${p.name} 的订阅？`)) return
  try {
    const data = await api<{ bytes: number }>(`/api/providers/${p.uuid}/fetch`, { method: 'POST' })
    toast.info(`「${p.name}」更新成功：${data.bytes} bytes`)
  } catch (e) {
    toast.error('更新失败: ' + (e as Error).message)
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
      toast.info('「' + p.name + '」上传成功')
      await load()
    } catch (e) {
      toast.error('上传失败: ' + (e as Error).message)
    }
  }
  input.click()
}

onMounted(load)
</script>

<template>
  <ContentPage>
    <div class="page-head">
      <h1 class="page-title">Provider 管理</h1>
      <Btn variant="primary" @click="openCreate">+ 添加</Btn>
    </div>

    <p v-if="providers.length === 0" class="empty-tip">暂无 Provider，点右上角“+ 添加”创建。</p>
    <div v-else class="pcard-list">
      <div v-for="p in providers" :key="p.uuid" class="pcard">
        <div class="pcard-head">
          <span class="pcard-name">{{ p.name }}</span>
          <span class="src-tag">{{ p.source }}</span>
          <span v-if="p.default" class="src-tag">默认</span>
        </div>
        <div class="pcard-src" :title="p.source === 'upload' ? (p.file_name || '') : p.url">
          {{ p.source === 'upload' ? (p.file_name || '已上传配置') : p.url }}
        </div>
        <div v-if="p.message" class="pcard-msg">{{ p.message }}</div>
        <div class="pcard-actions">
          <Btn v-if="!p.default" size="sm" @click="setDefault(p)">设为默认</Btn>
          <Btn size="sm" @click="openEdit(p)">编辑</Btn>
          <Btn size="sm" variant="primary" @click="p.source === 'upload' ? uploadSub(p) : fetchSub(p)">
            {{ p.source === 'upload' ? '上传' : '更新' }}
          </Btn>
          <Btn size="sm" @click="openVersions(p)">版本</Btn>
          <Btn size="sm" variant="danger" @click="remove(p)">删除</Btn>
        </div>
      </div>
    </div>

    <Modal v-if="showForm" :title="formTitle" @close="closeForm">
      <form @submit.prevent="submit">
        <Field label="名称">
          <input v-model="form.name" placeholder="机场名称" required />
        </Field>
        <Field label="来源">
          <select v-model="form.source">
            <option value="url">url</option>
            <option value="upload">upload</option>
          </select>
        </Field>
        <Field v-if="form.source === 'url'" label="订阅 URL（http/https）">
          <input v-model="form.url" placeholder="https://..." />
        </Field>
        <Field v-else label="上传配置文件">
          <input type="file" @change="onFileChange" />
        </Field>
        <Field label="备注">
          <input v-model="form.message" placeholder="可选" />
        </Field>
        <div class="form-actions">
          <Btn type="submit" variant="primary">{{ submitLabel }}</Btn>
          <Btn @click="closeForm">取消</Btn>
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
.empty-tip {
  margin: 0.4rem 0;
  font-size: 0.9rem;
  color: var(--text-faint);
}
/* Provider 卡片 */
.pcard-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.pcard {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
  padding: 0.85rem 1rem;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-card);
}
.pcard-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}
.pcard-name {
  font-size: 1rem;
  font-weight: var(--font-weight-6);
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-tag {
  flex-shrink: 0;
  padding: 2px 8px;
  font-size: 0.75rem;
  font-weight: var(--font-weight-5);
  color: var(--text-secondary);
  background: var(--neutral-soft);
  border-radius: var(--radius-pill);
}
.pcard-src {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pcard-msg {
  font-size: 0.8rem;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pcard-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.6rem;
  padding-top: 0.6rem;
  border-top: 1px solid var(--border);
}
.form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}

</style>
