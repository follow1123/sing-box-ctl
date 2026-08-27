export class ApiError extends Error {}

/** 统一 API 请求：解析 JSON 响应，错误信息从后端 error 字段提取 */
export async function api<T = unknown>(url: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(url, options)
  const text = await resp.text()
  let data: unknown = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    // 非 JSON 响应（如 /config 输出）
  }
  if (!resp.ok) {
    const msg =
      data && typeof data === 'object' && 'error' in data
        ? String((data as { error: unknown }).error)
        : 'HTTP ' + resp.status
    throw new ApiError(msg)
  }
  return data as T
}
