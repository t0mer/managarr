// web/src/lib/types.ts

export type ProviderKind = 'sonarr' | 'radarr' | 'lidarr' | 'jackett' | 'deluge' | 'plex' | 'emby' | 'jellyfin' | 'bazarr'
export type IssueStatus = 'open' | 'acknowledged' | 'resolved'
export type LogLevel = 'debug' | 'info' | 'warn' | 'warning' | 'error' | 'fatal'
export type NotifyProvider = 'shoutrrr' | 'greenapi' | 'whatsapp_web'

export interface Instance {
  id: string
  kind: ProviderKind
  name: string
  base_url: string
  enabled: boolean
  created_at: string
}

export interface LogEntry {
  id: number
  instance_id: string
  ts: string
  level: LogLevel
  message: string
  source: string
  raw?: string
}

export interface Issue {
  id: string
  fingerprint: string
  instance_id: string
  level: LogLevel
  message: string
  status: IssueStatus
  first_seen: string
  last_seen: string
  count: number
  impact_score: number
}

export interface MetricPoint {
  ts: string
  value: number
}

export interface MetricSeries {
  metric: string
  instance_id: string
  points: MetricPoint[]
}

export interface NotifyChannel {
  id: string
  name: string
  provider: NotifyProvider
  notify_on_success: boolean
  notify_on_failure: boolean
  enabled: boolean
  created_at: string
}

export interface BackupTarget {
  id: string
  name: string
  type: string
  retention_days: number
  enabled: boolean
  created_at: string
}

export interface Backup {
  id: string
  target_id: string
  instance_id: string
  ts: string
  size_bytes: number
  status: string
  location?: string
  error?: string
}

export interface SyncJob {
  id: string
  source_instance_id: string
  target_instance_id: string
  selectors: string[]
  schedule?: string
  enabled: boolean
  created_at: string
}

export interface SyncChange {
  field: string
  old_value: unknown
  new_value: unknown
}

export interface SyncPreview {
  changes: SyncChange[]
  count: number
}

export interface LidarrStats {
  artists_total: number
  albums_total: number
  queue_total: number
  missing_albums: number
}

export interface SonarrStats {
  series_total: number
  queue_total: number
  missing_episodes: number
}

export interface RadarrStats {
  movies_total: number
  movies_on_disk: number
  missing_movies: number
  queue_total: number
}

export interface JackettIndexer {
  id: string
  name: string
  configured: boolean
  monitored: boolean
  test_status: 'ok' | 'error' | 'skipped'
  test_error?: string
}

export interface JackettStats {
  indexers: JackettIndexer[]
  total: number
  configured: number
  ok: number
  error: number
}

export interface DelugeStats {
  download_rate: number
  upload_rate: number
  num_connections: number
  torrents: {
    total: number
    downloading: number
    seeding: number
    paused: number
    error: number
  }
}

export interface PlexLibrary {
  key: string
  title: string
  type: 'movie' | 'show'
  count?: number    // movie libraries
  shows?: number    // show libraries
  seasons?: number  // show libraries
  episodes?: number // show libraries
}

export interface PlexStats {
  server_name: string
  active_sessions: number
  libraries: PlexLibrary[]
}

export interface HealthResponse {
  status: string
  version: string
  db: string
}

export interface VersionResponse {
  version: string
  commit: string
  date: string
}
