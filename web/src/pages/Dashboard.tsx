// web/src/pages/Dashboard.tsx
import { useQuery } from '@tanstack/react-query'
import {
  LineChart, Line, XAxis, YAxis, Tooltip,
  ResponsiveContainer, Legend,
} from 'recharts'
import { api } from '../lib/api'
import type { Instance, Issue, MetricSeries } from '../lib/types'

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

// ── main page ─────────────────────────────────────────────────────────────────

export function Dashboard() {
  const { data: instances = [] } = useQuery<Instance[]>({
    queryKey: ['instances'],
    queryFn: () => api.instances.list(),
    refetchInterval: 60_000,
  })

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
    </div>
  )
}
