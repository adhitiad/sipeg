export interface ApiErrorInit {
  status: number
  code: string
  message: string
  requestId?: string
  fields?: Record<string, string>
  rows?: unknown
  cause?: unknown
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
  authenticated?: boolean
}
