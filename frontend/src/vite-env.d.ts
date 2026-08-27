/// <reference types="vite/client" />

// monaco-editor 0.56 的 json 语言包未附带类型声明，这里补充
declare module 'monaco-editor/language/json/monaco.contribution.js' {
  export interface JsonDiagnosticsOptions {
    validate?: boolean
    allowComments?: boolean
    schemas?: Array<{ uri: string; fileMatch: string[]; schema: object }>
  }
  export const jsonDefaults: {
    setDiagnosticsOptions(options: JsonDiagnosticsOptions): void
  }
}
