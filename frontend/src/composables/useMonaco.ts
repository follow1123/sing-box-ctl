import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import * as monaco from 'monaco-editor/editor/editor.api.js'
import { jsonDefaults } from 'monaco-editor/language/json/monaco.contribution.js'
import JsonWorker from 'monaco-editor/language/json/json.worker.js?worker'
import schema from '../assets/sing-box-schema.json'
import { isDark } from './useTheme'

// Monaco 只使用 JSON 语言，worker 也用 json 的
self.MonacoEnvironment = {
  getWorker: () => new JsonWorker(),
}

// 已知问题：控制台偶发输出 [createInstance] ... UNKNOWN service ICodeLensCache，
// 属 monaco 编辑器创建时的内部告警，不影响编辑/校验/补全，暂未定位，待后续验证。

// json 语言服务类型补丁（见 vite-env.d.ts）
type JsonDiagnosticsOptions = Parameters<typeof jsonDefaults.setDiagnosticsOptions>[0]

export function useMonaco() {
  const editorEl = ref<HTMLElement | null>(null)
  let editor: monaco.editor.IStandaloneCodeEditor | null = null
  let pendingValue = ''

  onMounted(async () => {
    if (!editorEl.value) return
    // 挂载 schema 补全/校验
    jsonDefaults.setDiagnosticsOptions({
      validate: true,
      allowComments: false,
      schemas: [
        {
          uri: 'https://sing-box.sagernet.org/schema.json',
          fileMatch: ['*'],
          schema: schema as object,
        },
      ],
    } as JsonDiagnosticsOptions)
    editor = monaco.editor.create(editorEl.value, {
      value: pendingValue,
      language: 'json',
      theme: isDark() ? 'vs-dark' : 'vs',      fontSize: 13,
      tabSize: 2,
      folding: true,
      minimap: { enabled: false },
      automaticLayout: true,
      scrollBeyondLastLine: false,
    })
  })

  onBeforeUnmount(() => {
    editor?.dispose()
    editor = null
  })

  // 主题切换联动编辑器
  watch(
    () => isDark(),
    (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'),
  )

  function setValue(value: string): void {
    if (editor) editor.setValue(value)
    else pendingValue = value
  }

  function getValue(): string {
    return editor ? editor.getValue() : pendingValue
  }

  function format(): void {
    editor?.getAction('editor.action.formatDocument')?.run()
  }

  return { editorEl, setValue, getValue, format }
}
