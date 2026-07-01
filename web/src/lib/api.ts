// web/src/lib/api.ts
import type {
  HealthResponse, VersionResponse,
  Instance,
  MetricSeries, NotifyChannel, BackupTarget, Backup,
  SyncJob, SyncPreview, PlexStats, DelugeStats, JackettStats,
  SonarrStats, RadarrStats, LidarrStats
} from './types'

const BASE = '/api/v1'

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(`${res.status}: ${text}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

const get = <T>(path: string) => request<T>('GET', path)
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body)
const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body)
const patch = <T>(path: string, body?: unknown) => request<T>('PATCH', path, body)
const del = (path: string) => request<void>('DELETE', path)

export const api = {
  health: () => fetch('/api/v1/health').then(r => r.json() as Promise<HealthResponse>),
  version: () => fetch('/version').then(r => r.json() as Promise<VersionResponse>),

  instances: {
    list: () => get<Instance[]>('/instances'),
    get: (id: string) => get<Instance>(`/instances/${id}`),
    create: (body: { kind: string; name: string; base_url: string; api_key?: string; username?: string; password?: string }) =>
      post<Instance>('/instances', body),
    update: (id: string, body: { name: string; base_url: string; api_key?: string; username?: string; password?: string }) =>
      put<Instance>(`/instances/${id}`, body),
    delete: (id: string) => del(`/instances/${id}`),
    test: (id: string) => post<{ ok: boolean; error?: string }>(`/instances/${id}/test`),
    setEnabled: (id: string, enabled: boolean) =>
      patch<void>(`/instances/${id}/enabled`, { enabled }),
  },

  metrics: {
    series: (params: { instance_id?: string; metric?: string; from?: string; to?: string }) => {
      const q = new URLSearchParams()
      if (params.instance_id) q.set('instance_id', params.instance_id)
      if (params.metric) q.set('metric', params.metric)
      if (params.from) q.set('from', params.from)
      if (params.to) q.set('to', params.to)
      return get<MetricSeries[]>(`/metrics/series?${q.toString()}`)
    },
  },

  notify: {
    list: () => get<NotifyChannel[]>('/notify/channels'),
    create: (body: {
      name: string; provider: string; config: Record<string, string>;
      notify_on_success: boolean; notify_on_failure: boolean; enabled: boolean
    }) => post<NotifyChannel>('/notify/channels', body),
    update: (id: string, body: unknown) => put<NotifyChannel>(`/notify/channels/${id}`, body),
    delete: (id: string) => del(`/notify/channels/${id}`),
    test: (body: unknown) => post<{ ok: boolean; error?: string }>('/notify/channels/test', body),
  },

  backup: {
    listTargets: () => get<BackupTarget[]>('/backup/targets'),
    createTarget: (body: { name: string; path: string; type?: string; retention_days?: number; enabled?: boolean }) =>
      post<BackupTarget>('/backup/targets', body),
    deleteTarget: (id: string) => del(`/backup/targets/${id}`),
    run: (body: { target_id: string; instance_id: string }) =>
      post<{ backup_id: string; status: string }>('/backup/run', body),
    listBackups: (targetId: string) => get<Backup[]>(`/backup/targets/${targetId}/backups`),
  },

  sync: {
    listJobs: () => get<SyncJob[]>('/sync/jobs'),
    createJob: (body: {
      source_instance_id: string; target_instance_id: string;
      selectors?: string[]; schedule?: string; enabled?: boolean
    }) => post<{ id: string }>('/sync/jobs', body),
    deleteJob: (id: string) => del(`/sync/jobs/${id}`),
    preview: (id: string) => post<SyncPreview>(`/sync/jobs/${id}/preview`),
    apply: (id: string) => post<{ applied: number }>(`/sync/jobs/${id}/apply`),
  },

  plex: {
    stats: (id: string) => get<PlexStats>(`/instances/${id}/plex/stats`),
  },

  deluge: {
    stats: (id: string) => get<DelugeStats>(`/instances/${id}/deluge/stats`),
  },

  jackett: {
    stats: (id: string) => get<JackettStats>(`/instances/${id}/jackett/stats`),
    setMonitored: (instanceId: string, indexerId: string, monitored: boolean) =>
      patch<void>(`/instances/${instanceId}/jackett/indexers/${encodeURIComponent(indexerId)}`, { monitored }),
  },

  sonarr: {
    stats: (id: string) => get<SonarrStats>(`/instances/${id}/sonarr/stats`),
  },

  radarr: {
    stats: (id: string) => get<RadarrStats>(`/instances/${id}/radarr/stats`),
  },

  lidarr: {
    stats: (id: string) => get<LidarrStats>(`/instances/${id}/lidarr/stats`),
  },
}
