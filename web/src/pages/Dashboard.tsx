// web/src/pages/Dashboard.tsx
import { useQuery } from '@tanstack/react-query'
import {
  LineChart, Line, XAxis, YAxis, Tooltip,
  ResponsiveContainer, Legend,
} from 'recharts'
import { Film, Tv, Activity, Download, Upload, ArrowDownUp } from 'lucide-react'
import { api } from '../lib/api'
import type { Instance, Issue, MetricSeries, PlexStats, DelugeStats } from '../lib/types'

// ── helpers ──────────────────────────────────────────────────────────────────

/** Kind → short label colour */
const kindColour: Record<string, string> = {
  sonarr: 'bg-blue-500',
  radarr: 'bg-yellow-500',
  lidarr: 'bg-purple-500',
  jackett: 'bg-orange-500',
  deluge: 'bg-green-500',
  plex: 'bg-orange-400',
  emby: 'bg-teal-500',
  jellyfin: 'bg-violet-500',
}

/** Merge MetricSeries[] into Recharts-friendly rows keyed by short timestamp */
function mergeSeriesForChart(
  series: MetricSeries[],
  _instances: Instance[],
): Array<Record<string, string | number>> {
  if (!series.length) return []

  // Build a map ts → { ts, [instanceId]: value }
  const byTs = new Map<string, Record<string, string | number>>()

  for (const s of series) {
    for (const pt of s.points ?? []) {
      const label = new Date(pt.ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      if (!byTs.has(pt.ts)) byTs.set(pt.ts, { ts: label })
      byTs.get(pt.ts)![s.instance_id] = pt.value
    }
  }

  // Sort by original ts
  const sorted = [...byTs.entries()].sort(([a], [b]) => (a < b ? -1 : 1))
  return sorted.map(([, row]) => row)
}

/** Pick a distinct colour per series line */
const LINE_COLOURS = [
  '#6366f1', '#10b981', '#f59e0b', '#ef4444',
  '#3b82f6', '#8b5cf6', '#14b8a6', '#f97316',
]

// ── sub-components ───────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  sub,
  accent,
}: {
  label: string
  value: string | number
  sub?: string
  accent?: string
}) {
  return (
    <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6 flex flex-col gap-1">
      <p className="text-sm opacity-60 font-medium">{label}</p>
      <p className={`text-3xl font-bold mt-1 ${accent ?? 'text-[var(--fg)]'}`}>{value}</p>
      {sub && <p className="text-xs opacity-50 mt-0.5">{sub}</p>}
    </div>
  )
}

function KindBadge({ kind }: { kind: string }) {
  const bg = kindColour[kind] ?? 'bg-gray-500'
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-white text-xs font-semibold uppercase tracking-wide ${bg}`}
    >
      {kind}
    </span>
  )
}

function InstanceTable({ instances }: { instances: Instance[] }) {
  if (!instances.length) {
    return <p className="text-sm opacity-50 py-4 text-center">No instances registered yet.</p>
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)] text-left">
            <th className="pb-2 pr-4 font-semibold opacity-60 whitespace-nowrap">Kind</th>
            <th className="pb-2 pr-4 font-semibold opacity-60">Name</th>
            <th className="pb-2 pr-4 font-semibold opacity-60">URL</th>
            <th className="pb-2 font-semibold opacity-60 text-center">Enabled</th>
          </tr>
        </thead>
        <tbody>
          {instances.map((inst) => (
            <tr
              key={inst.id}
              className="border-b border-[var(--border)] last:border-0 hover:bg-[var(--bg)] transition-colors"
            >
              <td className="py-2 pr-4 whitespace-nowrap">
                <KindBadge kind={inst.kind} />
              </td>
              <td className="py-2 pr-4 font-medium whitespace-nowrap">{inst.name}</td>
              <td className="py-2 pr-4 opacity-60 break-all max-w-xs">{inst.base_url}</td>
              <td className="py-2 text-center">
                <span
                  className={`inline-block w-2.5 h-2.5 rounded-full ${
                    inst.enabled ? 'bg-green-500' : 'bg-gray-400'
                  }`}
                  title={inst.enabled ? 'Enabled' : 'Disabled'}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function MetricChart({
  series,
  instances,
  isLoading,
}: {
  series: MetricSeries[]
  instances: Instance[]
  isLoading: boolean
}) {
  const data = mergeSeriesForChart(series, instances)
  const seriesIds = [...new Set(series.map((s) => s.instance_id))]

  // Map instance_id → display name
  const nameMap = Object.fromEntries(instances.map((i) => [i.id, i.name]))

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48 opacity-50 text-sm">
        Loading metric data…
      </div>
    )
  }

  if (!data.length) {
    return (
      <div className="flex items-center justify-center h-48 opacity-50 text-sm">
        No metric data yet
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={220}>
      <LineChart data={data} margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
        <XAxis
          dataKey="ts"
          tick={{ fontSize: 11, opacity: 0.6 }}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          tick={{ fontSize: 11, opacity: 0.6 }}
          tickLine={false}
          axisLine={false}
          width={36}
        />
        <Tooltip
          contentStyle={{
            background: 'var(--sidebar-bg)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            fontSize: 12,
          }}
        />
        <Legend
          iconType="circle"
          iconSize={8}
          formatter={(value) => nameMap[value] ?? value}
          wrapperStyle={{ fontSize: 12, paddingTop: 8 }}
        />
        {seriesIds.map((id, idx) => (
          <Line
            key={id}
            type="monotone"
            dataKey={id}
            name={id}
            stroke={LINE_COLOURS[idx % LINE_COLOURS.length]}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  )
}

function PlexLibraryGrid({ stats }: { stats: PlexStats }) {
  const movies = stats.libraries.filter((l) => l.type === 'movie')
  const shows = stats.libraries.filter((l) => l.type === 'show')

  return (
    <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">{stats.server_name || 'Plex'}</h2>
          <p className="text-xs opacity-50 mt-0.5">Library Statistics</p>
        </div>
        <div className="flex items-center gap-2 bg-green-500/10 text-green-500 rounded-lg px-3 py-1.5">
          <Activity size={14} />
          <span className="text-sm font-semibold">{stats.active_sessions}</span>
          <span className="text-xs opacity-80">active stream{stats.active_sessions !== 1 ? 's' : ''}</span>
        </div>
      </div>

      {/* Movie Libraries */}
      {movies.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold opacity-60 uppercase tracking-wider mb-3 flex items-center gap-2">
            <Film size={13} />
            Movie Libraries
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {movies.map((lib) => (
              <div
                key={lib.key}
                className="rounded-lg bg-[var(--bg)] border border-[var(--border)] p-3 flex flex-col gap-1"
              >
                <div className="flex items-center gap-1.5 text-orange-400">
                  <Film size={14} />
                  <span className="text-xs font-medium truncate opacity-80">{lib.title}</span>
                </div>
                <p className="text-2xl font-bold">{lib.count ?? 0}</p>
                <p className="text-xs opacity-40">movies</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TV Show Libraries */}
      {shows.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold opacity-60 uppercase tracking-wider mb-3 flex items-center gap-2">
            <Tv size={13} />
            TV Show Libraries
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {shows.map((lib) => (
              <div
                key={lib.key}
                className="rounded-lg bg-[var(--bg)] border border-[var(--border)] p-3 flex flex-col gap-2"
              >
                <div className="flex items-center gap-1.5 text-violet-400">
                  <Tv size={14} />
                  <span className="text-xs font-medium truncate opacity-80">{lib.title}</span>
                </div>
                <div className="grid grid-cols-3 gap-1 text-center">
                  <div>
                    <p className="text-lg font-bold">{lib.shows ?? 0}</p>
                    <p className="text-xs opacity-40">shows</p>
                  </div>
                  <div>
                    <p className="text-lg font-bold">{lib.seasons ?? 0}</p>
                    <p className="text-xs opacity-40">seasons</p>
                  </div>
                  <div>
                    <p className="text-lg font-bold">{lib.episodes ?? 0}</p>
                    <p className="text-xs opacity-40">episodes</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {movies.length === 0 && shows.length === 0 && (
        <p className="text-sm opacity-40 text-center py-4">No libraries found</p>
      )}
    </div>
  )
}

function PlexInstanceCard({ instance }: { instance: Instance }) {
  const { data: stats, isLoading, isError } = useQuery<PlexStats>({
    queryKey: ['plex-stats', instance.id],
    queryFn: () => api.plex.stats(instance.id),
    refetchInterval: 60_000,
    retry: 1,
  })

  if (isLoading) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6 animate-pulse">
        <div className="h-4 w-32 bg-[var(--border)] rounded mb-4" />
        <div className="h-24 bg-[var(--border)] rounded" />
      </div>
    )
  }

  if (isError || !stats) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6">
        <p className="text-sm opacity-50">
          Could not load Plex stats for <span className="font-medium">{instance.name}</span>
        </p>
      </div>
    )
  }

  return <PlexLibraryGrid stats={stats} />
}

// ── Deluge ───────────────────────────────────────────────────────────────────

function fmtRate(bytesPerSec: number): string {
  if (bytesPerSec >= 1_048_576) return `${(bytesPerSec / 1_048_576).toFixed(1)} MB/s`
  if (bytesPerSec >= 1_024) return `${(bytesPerSec / 1_024).toFixed(1)} KB/s`
  return `${Math.round(bytesPerSec)} B/s`
}

function DelugePanel({ stats, name }: { stats: DelugeStats; name: string }) {
  const { torrents } = stats
  return (
    <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6 space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">{name}</h2>
          <p className="text-xs opacity-50 mt-0.5">Deluge torrent client</p>
        </div>
        <div className="flex items-center gap-1.5 text-xs opacity-60">
          <ArrowDownUp size={13} />
          <span>{stats.num_connections} connections</span>
        </div>
      </div>

      {/* Transfer rates */}
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-lg bg-[var(--bg)] border border-[var(--border)] p-3 flex items-center gap-3">
          <div className="p-2 rounded-md bg-green-500/10 text-green-500">
            <Download size={16} />
          </div>
          <div>
            <p className="text-xs opacity-50">Download</p>
            <p className="text-lg font-bold">{fmtRate(stats.download_rate)}</p>
          </div>
        </div>
        <div className="rounded-lg bg-[var(--bg)] border border-[var(--border)] p-3 flex items-center gap-3">
          <div className="p-2 rounded-md bg-blue-500/10 text-blue-500">
            <Upload size={16} />
          </div>
          <div>
            <p className="text-xs opacity-50">Upload</p>
            <p className="text-lg font-bold">{fmtRate(stats.upload_rate)}</p>
          </div>
        </div>
      </div>

      {/* Torrent counts */}
      <div>
        <p className="text-xs font-semibold opacity-50 uppercase tracking-wider mb-2">Torrents</p>
        <div className="grid grid-cols-5 gap-2 text-center">
          {[
            { label: 'Total', value: torrents.total, colour: 'text-[var(--fg)]' },
            { label: 'Downloading', value: torrents.downloading, colour: 'text-green-500' },
            { label: 'Seeding', value: torrents.seeding, colour: 'text-blue-500' },
            { label: 'Paused', value: torrents.paused, colour: 'text-yellow-500' },
            { label: 'Error', value: torrents.error, colour: 'text-red-500' },
          ].map(({ label, value, colour }) => (
            <div key={label} className="rounded-lg bg-[var(--bg)] border border-[var(--border)] py-2 px-1">
              <p className={`text-xl font-bold ${colour}`}>{value}</p>
              <p className="text-xs opacity-50 mt-0.5">{label}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function DelugeInstanceCard({ instance }: { instance: Instance }) {
  const { data: stats, isLoading, isError } = useQuery<DelugeStats>({
    queryKey: ['deluge-stats', instance.id],
    queryFn: () => api.deluge.stats(instance.id),
    refetchInterval: 30_000,
    retry: 1,
  })

  if (isLoading) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6 animate-pulse">
        <div className="h-4 w-32 bg-[var(--border)] rounded mb-4" />
        <div className="h-24 bg-[var(--border)] rounded" />
      </div>
    )
  }

  if (isError || !stats) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6">
        <p className="text-sm opacity-50">
          Could not load Deluge stats for <span className="font-medium">{instance.name}</span>
        </p>
      </div>
    )
  }

  return <DelugePanel stats={stats} name={instance.name} />
}

// ── main page ─────────────────────────────────────────────────────────────────

export function Dashboard() {
  const { data: instances = [] } = useQuery<Instance[]>({
    queryKey: ['instances'],
    queryFn: () => api.instances.list(),
    refetchInterval: 60_000,
  })

  const plexInstances = instances.filter((i) => i.kind === 'plex' && i.enabled)
  const delugeInstances = instances.filter((i) => i.kind === 'deluge' && i.enabled)

  const { data: openIssues = [] } = useQuery<Issue[]>({
    queryKey: ['issues', 'open'],
    queryFn: () => api.issues.list('open'),
    refetchInterval: 60_000,
  })

  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: () => api.health(),
    refetchInterval: 60_000,
  })

  const { data: metricSeries = [], isLoading: metricsLoading } = useQuery<MetricSeries[]>({
    queryKey: ['metrics', 'queue_size'],
    queryFn: () => api.metrics.series({ metric: 'queue_size' }),
    refetchInterval: 60_000,
  })

  const totalInstances = instances.length
  const enabledInstances = instances.filter((i) => i.enabled).length
  const openIssueCount = openIssues.length

  const dbStatus = health?.db ?? '—'
  const dbOk = dbStatus === 'ok'

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-sm opacity-50 mt-0.5">Overview of your media stack</p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard label="Total Instances" value={totalInstances} />
        <StatCard
          label="Enabled Instances"
          value={enabledInstances}
          sub={`of ${totalInstances}`}
          accent={enabledInstances > 0 ? 'text-green-500' : undefined}
        />
        <StatCard
          label="Open Issues"
          value={openIssueCount}
          accent={openIssueCount > 0 ? 'text-red-500' : 'text-green-500'}
        />
        <StatCard
          label="DB Status"
          value={dbStatus}
          accent={dbOk ? 'text-green-500' : 'text-red-500'}
          sub={health?.version ? `v${health.version}` : undefined}
        />
      </div>

      {/* Metric chart */}
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6">
        <h2 className="text-base font-semibold mb-4">Queue Size</h2>
        <MetricChart
          series={metricSeries}
          instances={instances}
          isLoading={metricsLoading}
        />
      </div>

      {/* Instance table */}
      <div className="rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] p-6">
        <h2 className="text-base font-semibold mb-4">Registered Instances</h2>
        <InstanceTable instances={instances} />
      </div>

      {/* Deluge stats */}
      {delugeInstances.length > 0 && (
        <div className="space-y-4">
          <h2 className="text-base font-semibold">Deluge</h2>
          {delugeInstances.map((inst) => (
            <DelugeInstanceCard key={inst.id} instance={inst} />
          ))}
        </div>
      )}

      {/* Plex library stats */}
      {plexInstances.length > 0 && (
        <div className="space-y-4">
          <h2 className="text-base font-semibold">Plex Media Servers</h2>
          {plexInstances.map((inst) => (
            <PlexInstanceCard key={inst.id} instance={inst} />
          ))}
        </div>
      )}
    </div>
  )
}
