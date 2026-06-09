// web/src/pages/Backup.tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { BackupTarget, Backup, Instance } from '../lib/types'

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function formatTs(ts: string): string {
  return new Date(ts).toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function instanceName(instances: Instance[] | undefined, id: string): string {
  return instances?.find(i => i.id === id)?.name ?? id
}

// ─── StatusBadge ────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: string }) {
  const cls = status === 'success'
    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    : status === 'pending'
      ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
      : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'

  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      {status}
    </span>
  )
}

// ─── Modal ──────────────────────────────────────────────────────────────────

interface ModalProps {
  title: string
  onClose: () => void
  onSubmit: () => void
  submitLabel: string
  submitting?: boolean
  children: React.ReactNode
}

function Modal({ title, onClose, onSubmit, submitLabel, submitting, children }: ModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="w-full max-w-md rounded-xl bg-white dark:bg-gray-900 shadow-2xl">
        <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            aria-label="Close"
          >
            <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>
        <div className="px-6 py-5">{children}</div>
        <div className="flex justify-end gap-3 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
          <button
            onClick={onClose}
            disabled={submitting}
            className="rounded-md px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onSubmit}
            disabled={submitting}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Saving…' : submitLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Inline backups sub-table ────────────────────────────────────────────────

interface BackupsTableProps {
  targetId: string
  instances: Instance[] | undefined
}

function BackupsTable({ targetId, instances }: BackupsTableProps) {
  const { data: backups, isLoading, error } = useQuery({
    queryKey: ['backups', targetId],
    queryFn: () => api.backup.listBackups(targetId),
  })

  if (isLoading) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-4 text-sm text-gray-500 dark:text-gray-400">
          <span className="flex items-center gap-2">
            <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
            Loading backups…
          </span>
        </td>
      </tr>
    )
  }

  if (error) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-3 text-sm text-red-600 dark:text-red-400">
          Failed to load backups: {(error as Error).message}
        </td>
      </tr>
    )
  }

  if (!backups || backups.length === 0) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-4 text-sm text-gray-400 dark:text-gray-500 italic">
          No backups yet for this target.
        </td>
      </tr>
    )
  }

  return (
    <>
      {/* Sub-header */}
      <tr className="bg-gray-50 dark:bg-gray-800/60">
        <td colSpan={6} className="px-8 py-2">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-gray-500 dark:text-gray-400 uppercase tracking-wider font-semibold">
                <th className="pb-1 pr-4">Timestamp</th>
                <th className="pb-1 pr-4">Instance</th>
                <th className="pb-1 pr-4">Size</th>
                <th className="pb-1 pr-4">Status</th>
                <th className="pb-1">Location</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {backups.map((b: Backup) => (
                <tr key={b.id} className="text-gray-700 dark:text-gray-300">
                  <td className="py-1.5 pr-4 whitespace-nowrap">{formatTs(b.ts)}</td>
                  <td className="py-1.5 pr-4 whitespace-nowrap">{instanceName(instances, b.instance_id)}</td>
                  <td className="py-1.5 pr-4 whitespace-nowrap">{formatBytes(b.size_bytes)}</td>
                  <td className="py-1.5 pr-4 whitespace-nowrap">
                    <StatusBadge status={b.status} />
                    {b.error && (
                      <span className="ml-2 text-red-500 dark:text-red-400" title={b.error}>
                        {b.error.length > 40 ? b.error.slice(0, 40) + '…' : b.error}
                      </span>
                    )}
                  </td>
                  <td className="py-1.5 font-mono text-gray-500 dark:text-gray-400">
                    {b.location ?? '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </td>
      </tr>
    </>
  )
}

// ─── AddTargetForm ───────────────────────────────────────────────────────────

interface TargetFormState {
  name: string
  path: string
  retention_days: number
  enabled: boolean
}

function defaultTargetForm(): TargetFormState {
  return { name: '', path: '', retention_days: 30, enabled: true }
}

interface TargetFormProps {
  form: TargetFormState
  onChange: (f: TargetFormState) => void
  disabled?: boolean
}

function TargetForm({ form, onChange, disabled }: TargetFormProps) {
  const inputCls =
    'w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 ' +
    'px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 ' +
    'focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50'

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
        <input
          className={inputCls}
          type="text"
          placeholder="Daily backups"
          value={form.name}
          disabled={disabled}
          onChange={e => onChange({ ...form, name: e.target.value })}
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Path</label>
        <input
          className={inputCls}
          type="text"
          placeholder="/mnt/backups"
          value={form.path}
          disabled={disabled}
          onChange={e => onChange({ ...form, path: e.target.value })}
        />
        <p className="text-xs text-gray-400 dark:text-gray-500">Local filesystem path where backups will be stored.</p>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Retention (days)</label>
        <input
          className={inputCls}
          type="number"
          min={1}
          max={3650}
          value={form.retention_days}
          disabled={disabled}
          onChange={e => onChange({ ...form, retention_days: Math.max(1, parseInt(e.target.value, 10) || 30) })}
        />
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          role="switch"
          aria-checked={form.enabled}
          disabled={disabled}
          onClick={() => onChange({ ...form, enabled: !form.enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-1 disabled:opacity-50 ${
            form.enabled ? 'bg-indigo-600' : 'bg-gray-300 dark:bg-gray-600'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
              form.enabled ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
        <span className="text-sm text-gray-700 dark:text-gray-300">Enabled</span>
      </div>
    </div>
  )
}

// ─── RunBackupModal ──────────────────────────────────────────────────────────

interface RunBackupModalProps {
  target: BackupTarget
  instances: Instance[]
  onClose: () => void
}

function RunBackupModal({ target, instances, onClose }: RunBackupModalProps) {
  const qc = useQueryClient()
  const [instanceId, setInstanceId] = useState(instances[0]?.id ?? '')
  const [result, setResult] = useState<{ backup_id: string; status: string } | null>(null)

  const runMut = useMutation({
    mutationFn: () => api.backup.run({ target_id: target.id, instance_id: instanceId }),
    onSuccess: data => {
      setResult(data)
      // Invalidate so the backups sub-table refreshes if already open
      qc.invalidateQueries({ queryKey: ['backups', target.id] })
    },
  })

  const inputCls =
    'w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 ' +
    'px-3 py-2 text-sm text-gray-900 dark:text-gray-100 ' +
    'focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50'

  if (result) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        onClick={e => { if (e.target === e.currentTarget) onClose() }}
      >
        <div className="w-full max-w-md rounded-xl bg-white dark:bg-gray-900 shadow-2xl">
          <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Backup started</h2>
            <button
              onClick={onClose}
              className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              aria-label="Close"
            >
              <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
          <div className="px-6 py-5">
            <p className="text-sm text-gray-700 dark:text-gray-300">
              Backup enqueued successfully.
            </p>
            <dl className="mt-3 grid grid-cols-2 gap-2 text-sm">
              <dt className="font-medium text-gray-500 dark:text-gray-400">Backup ID</dt>
              <dd className="font-mono text-gray-900 dark:text-gray-100 break-all">{result.backup_id}</dd>
              <dt className="font-medium text-gray-500 dark:text-gray-400">Status</dt>
              <dd><StatusBadge status={result.status} /></dd>
            </dl>
          </div>
          <div className="flex justify-end border-t border-gray-200 dark:border-gray-700 px-6 py-4">
            <button
              onClick={onClose}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="w-full max-w-md rounded-xl bg-white dark:bg-gray-900 shadow-2xl">
        <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Run Backup — {target.name}
          </h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            aria-label="Close"
          >
            <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>
        <div className="px-6 py-5">
          {instances.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">
              No instances available. Add an app first.
            </p>
          ) : (
            <div className="flex flex-col gap-3">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Select instance to back up
              </label>
              <select
                className={inputCls}
                value={instanceId}
                disabled={runMut.isPending}
                onChange={e => setInstanceId(e.target.value)}
              >
                {instances.map(inst => (
                  <option key={inst.id} value={inst.id}>
                    {inst.name} ({inst.kind})
                  </option>
                ))}
              </select>
              {runMut.isError && (
                <p className="text-sm text-red-600 dark:text-red-400">
                  {(runMut.error as Error).message}
                </p>
              )}
            </div>
          )}
        </div>
        <div className="flex justify-end gap-3 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
          <button
            onClick={onClose}
            disabled={runMut.isPending}
            className="rounded-md px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => runMut.mutate()}
            disabled={runMut.isPending || instances.length === 0}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {runMut.isPending ? 'Starting…' : 'Run Backup'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Backup page ─────────────────────────────────────────────────────────────

export function Backup() {
  const qc = useQueryClient()

  const { data: targets, isLoading, error } = useQuery({
    queryKey: ['backup-targets'],
    queryFn: api.backup.listTargets,
  })

  const { data: instances } = useQuery({
    queryKey: ['instances'],
    queryFn: api.instances.list,
  })

  // Add target dialog
  const [showAdd, setShowAdd] = useState(false)
  const [addForm, setAddForm] = useState<TargetFormState>(defaultTargetForm())

  // Run backup modal — holds the target to run against
  const [runTarget, setRunTarget] = useState<BackupTarget | null>(null)

  // Expanded backups per target id
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  function toggleExpanded(id: string) {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function invalidate() {
    qc.invalidateQueries({ queryKey: ['backup-targets'] })
  }

  const createMut = useMutation({
    mutationFn: (f: TargetFormState) =>
      api.backup.createTarget({
        name: f.name,
        path: f.path,
        retention_days: f.retention_days,
        enabled: f.enabled,
      }),
    onSuccess: () => {
      invalidate()
      setShowAdd(false)
      setAddForm(defaultTargetForm())
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.backup.deleteTarget(id),
    onSuccess: invalidate,
  })

  function handleDelete(t: BackupTarget) {
    if (window.confirm(`Delete target "${t.name}"? This cannot be undone.`)) {
      deleteMut.mutate(t.id)
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Backup</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage backup targets and run configuration backups for your apps
          </p>
        </div>
        <button
          onClick={() => { setAddForm(defaultTargetForm()); setShowAdd(true) }}
          className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
          Add Target
        </button>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex items-center justify-center py-20 text-gray-500 dark:text-gray-400">
          <svg className="mr-3 h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
          </svg>
          Loading…
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load backup targets: {(error as Error).message}
        </div>
      )}

      {/* Empty state */}
      {targets && targets.length === 0 && (
        <div className="rounded-xl border-2 border-dashed border-gray-200 dark:border-gray-700 py-20 text-center">
          <svg
            className="mx-auto h-12 w-12 text-gray-300 dark:text-gray-600"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">No backup targets yet</p>
          <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">Click "Add Target" to create one.</p>
        </div>
      )}

      {/* Targets table */}
      {targets && targets.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                {['Name', 'Type', 'Retention', 'Status', 'Actions'].map(h => (
                  <th
                    key={h}
                    className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-800 bg-white dark:bg-gray-900">
              {targets.map(target => {
                const isDeleting = deleteMut.isPending && deleteMut.variables === target.id
                const isExpanded = expanded.has(target.id)

                return (
                  <>
                    <tr
                      key={target.id}
                      className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                    >
                      {/* Name */}
                      <td className="px-4 py-3 whitespace-nowrap font-medium text-gray-900 dark:text-gray-100">
                        {target.name}
                      </td>

                      {/* Type */}
                      <td className="px-4 py-3 whitespace-nowrap text-gray-500 dark:text-gray-400 font-mono text-xs">
                        {target.type || 'local'}
                      </td>

                      {/* Retention */}
                      <td className="px-4 py-3 whitespace-nowrap text-gray-500 dark:text-gray-400">
                        {target.retention_days}d
                      </td>

                      {/* Status */}
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span
                          className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            target.enabled
                              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                              : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                          }`}
                        >
                          <span
                            className={`h-1.5 w-1.5 rounded-full ${target.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
                          />
                          {target.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                      </td>

                      {/* Actions */}
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap items-center gap-2">
                          {/* Run backup */}
                          <button
                            onClick={() => setRunTarget(target)}
                            disabled={isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 disabled:opacity-50 transition-colors"
                          >
                            Run Backup
                          </button>

                          {/* View Backups toggle */}
                          <button
                            onClick={() => toggleExpanded(target.id)}
                            disabled={isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
                          >
                            {isExpanded ? 'Hide Backups' : 'View Backups'}
                          </button>

                          {/* Delete */}
                          <button
                            onClick={() => handleDelete(target)}
                            disabled={isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 disabled:opacity-50 transition-colors"
                          >
                            {isDeleting ? 'Deleting…' : 'Delete'}
                          </button>
                        </div>
                      </td>
                    </tr>

                    {/* Inline backups sub-table */}
                    {isExpanded && (
                      <BackupsTable
                        key={`backups-${target.id}`}
                        targetId={target.id}
                        instances={instances}
                      />
                    )}
                  </>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Add target dialog ── */}
      {showAdd && (
        <Modal
          title="Add Backup Target"
          onClose={() => setShowAdd(false)}
          onSubmit={() => createMut.mutate(addForm)}
          submitLabel="Add Target"
          submitting={createMut.isPending}
        >
          <TargetForm form={addForm} onChange={setAddForm} disabled={createMut.isPending} />
          {createMut.isError && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {(createMut.error as Error).message}
            </p>
          )}
        </Modal>
      )}

      {/* ── Run backup modal ── */}
      {runTarget && (
        <RunBackupModal
          target={runTarget}
          instances={instances ?? []}
          onClose={() => setRunTarget(null)}
        />
      )}
    </div>
  )
}
