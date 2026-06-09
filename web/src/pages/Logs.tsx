// web/src/pages/Logs.tsx
import { useEffect, useRef, useCallback, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { LogEntry, LogLevel, Instance } from '../lib/types'

// ── Level badge ──────────────────────────────────────────────────────────────

const LEVEL_CLASSES: Record<LogLevel, string> = {
  debug:   'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300',
  info:    'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  warn:    'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  warning: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  error:   'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  fatal:   'bg-red-900 text-red-100 dark:bg-red-950 dark:text-red-200',
}

function LevelBadge({ level }: { level: LogLevel }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold uppercase ${LEVEL_CLASSES[level] ?? LEVEL_CLASSES.info}`}>
      {level}
    </span>
  )
}

// ── Timestamp formatting ──────────────────────────────────────────────────────

function formatTs(ts: string): string {
  const d = new Date(ts)
  if (isNaN(d.getTime())) return ts
  const now = Date.now()
  const diffMs = now - d.getTime()
  if (diffMs < 60_000) return `${Math.floor(diffMs / 1000)}s ago`
  if (diffMs < 3_600_000) return `${Math.floor(diffMs / 60_000)}m ago`
  if (diffMs < 86_400_000) return `${Math.floor(diffMs / 3_600_000)}h ago`
  return d.toLocaleString()
}

// ── Log table ─────────────────────────────────────────────────────────────────

const LEVELS: Array<LogLevel | 'all'> = ['all', 'debug', 'info', 'warn', 'error', 'fatal']

function instanceName(instances: Instance[] | undefined, id: string): string {
  return instances?.find(i => i.id === id)?.name ?? id
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function Logs() {
  const [instanceFilter, setInstanceFilter] = useState<string>('')
  const [levelFilter, setLevelFilter]       = useState<string>('all')
  const [live, setLive]                     = useState(false)
  const [liveEntries, setLiveEntries]       = useState<LogEntry[]>([])
  const [sseError, setSseError]             = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Fetch instances for the filter dropdown
  const { data: instances } = useQuery({
    queryKey: ['instances'],
    queryFn: () => api.instances.list(),
  })

  // Historical log query (only when live=off)
  const { data: historicalLogs, isLoading } = useQuery({
    queryKey: ['logs', instanceFilter, levelFilter],
    queryFn: () =>
      api.logs.list({
        instance_id: instanceFilter || undefined,
        level:       levelFilter !== 'all' ? levelFilter : undefined,
        limit:       500,
      }),
    enabled: !live,
  })

  // Build SSE URL when live is on
  const streamUrl = live
    ? api.logs.streamUrl({
        instance_id: instanceFilter || undefined,
        level:       levelFilter !== 'all' ? levelFilter : undefined,
      })
    : null

  // Stable callback for SSE entries
  const handleEntry = useCallback((entry: LogEntry) => {
    setSseError(false)
    setLiveEntries(prev => {
      // cap at 2000 entries to avoid unbounded growth
      const next = [...prev, entry]
      return next.length > 2000 ? next.slice(next.length - 2000) : next
    })
  }, [])

  // Wire up SSE
  useEffect(() => {
    if (!streamUrl) return
    setSseError(false)
    const es = new EventSource(streamUrl)
    es.onmessage = (e) => {
      try {
        const entry = JSON.parse(e.data) as LogEntry
        handleEntry(entry)
      } catch {
        // ignore parse errors
      }
    }
    es.onerror = () => {
      setSseError(true)
      es.close()
    }
    return () => es.close()
  }, [streamUrl, handleEntry])

  // Auto-scroll when live entries change
  useEffect(() => {
    if (live && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [live, liveEntries])

  // When toggling live on, clear previous live entries
  function toggleLive() {
    if (!live) {
      setLiveEntries([])
      setSseError(false)
    }
    setLive(v => !v)
  }

  // When filters change while in live mode, reset entries
  function applyInstanceFilter(val: string) {
    setInstanceFilter(val)
    if (live) { setLiveEntries([]); setSseError(false) }
  }
  function applyLevelFilter(val: string) {
    setLevelFilter(val)
    if (live) { setLiveEntries([]); setSseError(false) }
  }

  const rows = live ? liveEntries : (historicalLogs ?? [])

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Logs</h1>
        <span className="text-sm opacity-50">{rows.length} entries</span>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-3 items-center p-3 rounded-lg border border-[var(--border)] bg-[var(--sidebar-bg)]">
        {/* Instance filter */}
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium opacity-70" htmlFor="log-instance">
            Instance
          </label>
          <select
            id="log-instance"
            value={instanceFilter}
            onChange={e => applyInstanceFilter(e.target.value)}
            className="text-sm rounded-md border border-[var(--border)] bg-[var(--bg)] px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All</option>
            {instances?.map(inst => (
              <option key={inst.id} value={inst.id}>{inst.name}</option>
            ))}
          </select>
        </div>

        {/* Level filter */}
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium opacity-70" htmlFor="log-level">
            Level
          </label>
          <select
            id="log-level"
            value={levelFilter}
            onChange={e => applyLevelFilter(e.target.value)}
            className="text-sm rounded-md border border-[var(--border)] bg-[var(--bg)] px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {LEVELS.map(l => (
              <option key={l} value={l}>{l === 'all' ? 'All levels' : l}</option>
            ))}
          </select>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* SSE error notice */}
        {sseError && (
          <span className="text-xs text-red-500 font-medium">
            Stream disconnected
          </span>
        )}

        {/* Live toggle */}
        <button
          onClick={toggleLive}
          className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
            live
              ? 'bg-red-500 text-white hover:bg-red-600'
              : 'bg-blue-500 text-white hover:bg-blue-600'
          }`}
        >
          {live ? (
            <>
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-white opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-white" />
              </span>
              Disconnect
            </>
          ) : (
            <>
              <span className="h-2 w-2 rounded-full bg-white" />
              Go Live
            </>
          )}
        </button>
      </div>

      {/* Table area */}
      <div className="flex-1 overflow-auto rounded-lg border border-[var(--border)]">
        {!live && isLoading ? (
          <div className="flex items-center justify-center py-20 opacity-50 text-sm">
            Loading logs…
          </div>
        ) : rows.length === 0 ? (
          <div className="flex items-center justify-center py-20 opacity-50 text-sm">
            {live ? 'Waiting for log entries…' : 'No log entries found.'}
          </div>
        ) : (
          <table className="w-full text-sm border-collapse">
            <thead className="sticky top-0 bg-[var(--sidebar-bg)] z-10">
              <tr className="text-left text-xs uppercase opacity-60 border-b border-[var(--border)]">
                <th className="px-3 py-2 whitespace-nowrap">Time</th>
                <th className="px-3 py-2 whitespace-nowrap">Instance</th>
                <th className="px-3 py-2 whitespace-nowrap">Level</th>
                <th className="px-3 py-2">Message</th>
                <th className="px-3 py-2 whitespace-nowrap hidden md:table-cell">Source</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((entry, idx) => (
                <tr
                  key={`${entry.id}-${idx}`}
                  className="border-b border-[var(--border)] hover:bg-[var(--sidebar-bg)] transition-colors"
                >
                  <td className="px-3 py-1.5 whitespace-nowrap opacity-60 tabular-nums text-xs">
                    {formatTs(entry.ts)}
                  </td>
                  <td className="px-3 py-1.5 whitespace-nowrap text-xs">
                    {instanceName(instances, entry.instance_id)}
                  </td>
                  <td className="px-3 py-1.5 whitespace-nowrap">
                    <LevelBadge level={entry.level} />
                  </td>
                  <td className="px-3 py-1.5 max-w-xl break-words font-mono text-xs">
                    {entry.message}
                  </td>
                  <td className="px-3 py-1.5 whitespace-nowrap opacity-50 text-xs hidden md:table-cell">
                    {entry.source}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {/* Scroll anchor */}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
