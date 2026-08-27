/** 与 Go 后端 API 数据结构对应 */

export interface ProviderConfig {
  uuid: string
  name: string
  url: string
  source: 'url' | 'upload'
  message?: string
  file_name?: string
}

export interface ProviderRequest {
  name: string
  url: string
  source: 'url' | 'upload'
  message?: string
}

export interface TemplateInfo {
  uuid: string
  name: string
  default: boolean
}

export interface UploadResult {
  ok: boolean
  bytes: number
  file_name: string
}
