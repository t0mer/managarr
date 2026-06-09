// web/src/lib/api.ts
import type { HealthResponse, VersionResponse } from './types'

const BASE = '/api/v1'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json() as Promise<T>
}

export const api = {
  health: () => get<HealthResponse>('/health'),
  version: () => fetch('/version').then(r => r.json() as Promise<VersionResponse>),
}
