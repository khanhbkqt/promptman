import type { ApiResponse } from '@/types/daemon'

let _baseUrl = ''
let _token = ''

export function configureApiClient(baseUrl: string, token: string) {
  _baseUrl = baseUrl.replace(/\/$/, '')
  _token = token
}

export async function apiRequest<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<ApiResponse<T>> {
  const url = `${_baseUrl}${path}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (_token) {
    headers['Authorization'] = `Bearer ${_token}`
  }

  try {
    const response = await fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    })

    const envelope: ApiResponse<T> = await response.json()
    return envelope
  } catch (error) {
    return {
      ok: false,
      data: null,
      error: {
        code: 'NETWORK_ERROR',
        message: error instanceof Error ? error.message : 'Network request failed',
      },
    }
  }
}

// Convenience methods
export const api = {
  get: <T>(path: string) => apiRequest<T>('GET', path),
  post: <T>(path: string, body?: unknown) => apiRequest<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => apiRequest<T>('PUT', path, body),
  delete: <T>(path: string) => apiRequest<T>('DELETE', path),
}
