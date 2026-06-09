// web/src/pages/Issues.tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { Issue, IssueStatus, LogLevel } from '../lib/types'

// ── helpers ────────────────────────────────────────────────────────────────

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60_000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  return `${days}d ago`
}

function levelBadgeClass(level: LogLevel): string {
  switch (level) {
    case 'debug':   return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    case 'info':    return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
    case 'warn':
    case 'warning': return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300'
    case 'error':
    case 'fatal':   return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
    default:        return 'bg-gray-100 text-gray-600'
  }
}

function impactClass(score: number): string {
  if (score <= 33) return 'text-green-600 dark:text-green-400'
  if (score <= 66) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
}

function impactBarClass(score: number): string {
  if (score <= 33) return 'bg-green-500'
  if (score <= 66) return 'bg-yellow-500'
  return 'bg-red-500'
}

// ── sub-components ──────────────────────────────────────────────────────────

interface ActionButtonsProps {
  issue: Issue
  onUpdate: (id: string, status: IssueStatus) => void
  isPending: boolean
}

function ActionButtons({ issue, onUpdate, isPending }: ActionButtonsProps) {
  const btnBase =
    'px-2.5 py-1 rounded text-xs font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed'

  return (
    <div className="flex gap-1.5 justify-end" onClick={(e) => e.stopPropagation()}>
      {(issue.status === 'open') && (
        <button
          className={`${btnBase} bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/40 dark:text-yellow-300 dark:hover:bg-yellow-900/60`}
          disabled={isPending}
          onClick={() => onUpdate(issue.id, 'acknowledged')}
        >
          Acknowledge
        </button>
      )}
      {(issue.status === 'open' || issue.status === 'acknowledged') && (
        <button
          className={`${btnBase} bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/40 dark:text-green-300 dark:hover:bg-green-900/60`}
          disabled={isPending}
          onClick={() => onUpdate(issue.id, 'resolved')}
        >
          Resolve
        </button>
      )}
      {(issue.status === 'acknowledged' || issue.status === 'resolved') && (
        <button
          className={`${btnBase} bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/40 dark:text-blue-300 dark:hover:bg-blue-900/60`}
          disabled={isPending}
          onClick={() => onUpdate(issue.id, 'open')}
        >
          Reopen
        </button>
      )}
    </div>
  )
}

interface IssueRowProps {
  issue: Issue
  expanded: boolean
  onToggle: () => void
  onUpdate: (id: string, status: IssueStatus) => void
  isPending: boolean
}

function IssueRow({ issue, expanded, onToggle, onUpdate, isPending }: IssueRowProps) {
  return (
    <>
      <tr
        className="border-b border-[var(--border)] hover:bg-[var(--sidebar-bg)] cursor-pointer transition-colors"
        onClick={onToggle}
      >
        {/* Level */}
        <td className="px-4 py-3 whitespace-nowrap">
          <span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold uppercase tracking-wide ${levelBadgeClass(issue.level)}`}>
            {issue.level}
          </span>
        </td>

        {/* Message */}
        <td className="px-4 py-3 max-w-xs lg:max-w-sm xl:max-w-lg">
          <p className="truncate text-sm font-medium" title={issue.message}>
            {issue.message}
          </p>
        </td>

        {/* Instance */}
        <td className="px-4 py-3 whitespace-nowrap text-sm opacity-70">
          {issue.instance_id}
        </td>

        {/* Count */}
        <td className="px-4 py-3 whitespace-nowrap text-sm text-center font-mono">
          {issue.count.toLocaleString()}
        </td>

        {/* Impact score */}
        <td className="px-4 py-3 whitespace-nowrap text-sm">
          <div className="flex items-center gap-2 min-w-[80px]">
            <div className="flex-1 h-1.5 rounded-full bg-gray-200 dark:bg-gray-700">
              <div
                className={`h-full rounded-full ${impactBarClass(issue.impact_score)}`}
                style={{ width: `${issue.impact_score}%` }}
              />
            </div>
            <span className={`font-semibold text-xs w-6 text-right ${impactClass(issue.impact_score)}`}>
              {issue.impact_score}
            </span>
          </div>
        </td>

        {/* Last seen */}
        <td className="px-4 py-3 whitespace-nowrap text-sm opacity-70">
          {relativeTime(issue.last_seen)}
        </td>

        {/* Actions */}
        <td className="px-4 py-3 whitespace-nowrap text-right">
          <ActionButtons issue={issue} onUpdate={onUpdate} isPending={isPending} />
        </td>
      </tr>

      {/* Expanded detail row */}
      {expanded && (
        <tr className="border-b border-[var(--border)] bg-[var(--sidebar-bg)]">
          <td colSpan={7} className="px-6 py-3">
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-x-8 gap-y-1 text-xs">
              <div>
                <dt className="opacity-50 uppercase tracking-wide font-medium">Fingerprint</dt>
                <dd className="font-mono mt-0.5 break-all">{issue.fingerprint}</dd>
              </div>
              <div>
                <dt className="opacity-50 uppercase tracking-wide font-medium">First seen</dt>
                <dd className="mt-0.5">
                  {new Date(issue.first_seen).toLocaleString()} ({relativeTime(issue.first_seen)})
                </dd>
              </div>
              <div>
                <dt className="opacity-50 uppercase tracking-wide font-medium">Issue ID</dt>
                <dd className="font-mono mt-0.5 break-all">{issue.id}</dd>
              </div>
            </dl>
          </td>
        </tr>
      )}
    </>
  )
}

// ── main page ───────────────────────────────────────────────────────────────

const STATUS_TABS: { label: string; value: IssueStatus }[] = [
  { label: 'Open',         value: 'open' },
  { label: 'Acknowledged', value: 'acknowledged' },
  { label: 'Resolved',     value: 'resolved' },
]

export function Issues() {
  const [activeStatus, setActiveStatus] = useState<IssueStatus>('open')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const qc = useQueryClient()

  const { data: issues = [], isLoading, isError } = useQuery({
    queryKey: ['issues', activeStatus],
    queryFn: () => api.issues.list(activeStatus),
  })

  const updateStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: IssueStatus }) =>
      api.issues.updateStatus(id, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['issues'] })
    },
  })

  function handleUpdate(id: string, status: IssueStatus) {
    updateStatus.mutate({ id, status })
  }

  function toggleExpand(id: string) {
    setExpandedId((prev) => (prev === id ? null : id))
  }

  return (
    <div className="space-y-4">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-semibold">Issues</h1>
        <p className="text-sm opacity-60 mt-0.5">
          Deduplicated and ranked issues from your monitored apps.
        </p>
      </div>

      {/* Status filter tabs */}
      <div className="flex gap-1 border-b border-[var(--border)] pb-0">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            className={[
              'px-4 py-2 text-sm font-medium rounded-t transition-colors -mb-px border-b-2',
              activeStatus === tab.value
                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                : 'border-transparent opacity-60 hover:opacity-90',
            ].join(' ')}
            onClick={() => {
              setActiveStatus(tab.value)
              setExpandedId(null)
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="rounded-xl border border-[var(--border)] overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20 text-sm opacity-50">
            Loading issues…
          </div>
        ) : isError ? (
          <div className="flex items-center justify-center py-20 text-sm text-red-500">
            Failed to load issues. Check the server connection.
          </div>
        ) : issues.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 gap-2 text-sm opacity-50">
            <span className="text-3xl">&#10003;</span>
            <span>No {activeStatus} issues found.</span>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead className="text-xs uppercase tracking-wide opacity-60 bg-[var(--sidebar-bg)] border-b border-[var(--border)]">
                <tr>
                  <th className="px-4 py-3 font-medium">Level</th>
                  <th className="px-4 py-3 font-medium">Message</th>
                  <th className="px-4 py-3 font-medium">Instance</th>
                  <th className="px-4 py-3 font-medium text-center">Count</th>
                  <th className="px-4 py-3 font-medium">Impact</th>
                  <th className="px-4 py-3 font-medium">Last Seen</th>
                  <th className="px-4 py-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {issues.map((issue) => (
                  <IssueRow
                    key={issue.id}
                    issue={issue}
                    expanded={expandedId === issue.id}
                    onToggle={() => toggleExpand(issue.id)}
                    onUpdate={handleUpdate}
                    isPending={updateStatus.isPending}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Summary line */}
      {!isLoading && !isError && issues.length > 0 && (
        <p className="text-xs opacity-40 text-right">
          {issues.length} {activeStatus} issue{issues.length !== 1 ? 's' : ''}
        </p>
      )}
    </div>
  )
}
