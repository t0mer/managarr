// web/src/lib/types.ts

export type ProviderKind =
  | 'sonarr' | 'radarr' | 'lidarr'
  | 'jackett' | 'deluge'
  | 'plex' | 'emby' | 'jellyfin'

export interface Instance {
  id: string
  kind: ProviderKind
  name: string
  baseUrl: string
  enabled: boolean
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
