import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, XCircle, MinusCircle } from 'lucide-react'
import { api } from '../lib/api'
import type { Instance, ProviderKind, JackettStats, JackettIndexer } from '../lib/types'

const KIND_COLORS: Record<ProviderKind, string> = {
  sonarr: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  radarr: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
  lidarr: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
  jackett: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  deluge: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
  plex: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  emby: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200',
  jellyfin: 'bg-pink-100 text-pink-800 dark:bg-pink-900 dark:text-pink-200',
}

const KINDS: ProviderKind[] = [
  'sonarr', 'radarr', 'lidarr', 'jackett', 'deluge', 'plex', 'emby', 'jellyfin',
]

interface FormState {
  kind: ProviderKind
  name: string
  base_url: string
  api_key: string
}

function defaultForm(): FormState {
  return { kind: 'sonarr', name: '', base_url: '', api_key: '' }
}

// ─── AppForm ────────────────────────────────────────────────────────────────

interface AppFormProps {
  form: FormState
  onChange: (f: FormState) => void
  showKind: boolean
  disabled?: boolean
}

function AppForm({ form, onChange, showKind, disabled }: AppFormProps) {
  const inputCls =
    'w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 ' +
    'px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 ' +
    'focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50'

  return (
    <div className="flex flex-col gap-4">
      {showKind && (
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Kind</label>
          <select
            className={inputCls}
            value={form.kind}
            disabled={disabled}
            onChange={e => onChange({ ...form, kind: e.target.value as ProviderKind })}
          >
            {KINDS.map(k => (
              <option key={k} value={k}>
                {k.charAt(0).toUpperCase() + k.slice(1)}
              </option>
            ))}
          </select>
        </div>
      )}

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
        <input
          className={inputCls}
          type="text"
          placeholder="My Sonarr"
          value={form.name}
          disabled={disabled}
          onChange={e => onChange({ ...form, name: e.target.value })}
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Base URL</label>
        <input
          className={inputCls}
          type="url"
          placeholder="http://192.168.1.10:8989"
          value={form.base_url}
          disabled={disabled}
          onChange={e => onChange({ ...form, base_url: e.target.value })}
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          API Key <span className="font-normal text-gray-400">(optional)</span>
        </label>
        <input
          className={inputCls}
          type="password"
          placeholder="leave blank to keep existing"
          value={form.api_key}
          disabled={disabled}
          onChange={e => onChange({ ...form, api_key: e.target.value })}
          autoComplete="new-password"
        />
      </div>
    </div>
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
        {/* Header */}
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

        {/* Body */}
        <div className="px-6 py-5">{children}</div>

        {/* Footer */}
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

// ─── Jackett indexer modal ──────────────────────────────────────────────────

function IndexerStatusIcon({ status }: { status: JackettIndexer['test_status'] }) {
  if (status === 'ok') return <CheckCircle2 size={14} className="text-green-500 shrink-0" />
  if (status === 'error') return <XCircle size={14} className="text-red-500 shrink-0" />
  return <MinusCircle size={14} className="text-gray-400 shrink-0" />
}

function JackettIndexersModal({ instance, onClose }: { instance: Instance; onClose: () => void }) {
  const qc = useQueryClient()

  const { data: stats, isLoading, isError, refetch, isFetching } = useQuery<JackettStats>({
    queryKey: ['jackett-stats', instance.id],
    queryFn: () => api.jackett.stats(instance.id),
    retry: 1,
  })

  const toggleMut = useMutation({
    mutationFn: ({ indexerId, monitored }: { indexerId: string; monitored: boolean }) =>
      api.jackett.setMonitored(instance.id, indexerId, monitored),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['jackett-stats', instance.id] }),
  })

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="w-full max-w-2xl rounded-xl bg-white dark:bg-gray-900 shadow-2xl flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4 shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Jackett Indexers — {instance.name}
            </h2>
            {stats && (
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                {stats.configured} configured · {stats.ok} OK
                {stats.error > 0 && ` · ${stats.error} error`}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="rounded px-3 py-1.5 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 disabled:opacity-50 transition-colors"
            >
              {isFetching ? 'Testing…' : 'Re-test all'}
            </button>
            <button
              onClick={onClose}
              className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="overflow-y-auto px-6 py-4">
          {isLoading && (
            <div className="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400 gap-2 text-sm">
              <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              Testing indexers…
            </div>
          )}
          {isError && (
            <p className="text-sm text-red-600 dark:text-red-400 py-8 text-center">
              Could not fetch indexers. Check that the Jackett instance is reachable.
            </p>
          )}
          {stats && stats.indexers.length === 0 && (
            <p className="text-sm text-gray-400 dark:text-gray-500 py-8 text-center">No indexers found.</p>
          )}
          {stats && stats.indexers.length > 0 && (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-700">
                  <th className="pb-2 pr-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400">Status</th>
                  <th className="pb-2 pr-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400">Indexer</th>
                  <th className="pb-2 pr-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400">Configured</th>
                  <th className="pb-2 text-right text-xs font-semibold text-gray-500 dark:text-gray-400">Monitor</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                {stats.indexers.map((idx) => (
                  <tr key={idx.id} className={idx.monitored ? '' : 'opacity-40'}>
                    <td className="py-2 pr-3">
                      <div title={idx.test_error ?? idx.test_status}>
                        <IndexerStatusIcon status={idx.test_status} />
                      </div>
                    </td>
                    <td className="py-2 pr-3">
                      <p className="font-medium text-gray-900 dark:text-gray-100">{idx.name || idx.id}</p>
                      {idx.test_status === 'error' && idx.test_error && (
                        <p className="text-xs text-red-500 mt-0.5 truncate max-w-xs">{idx.test_error}</p>
                      )}
                    </td>
                    <td className="py-2 pr-3">
                      {idx.configured
                        ? <span className="text-xs text-green-600 dark:text-green-400">Yes</span>
                        : <span className="text-xs text-gray-400">No</span>}
                    </td>
                    <td className="py-2 text-right">
                      <button
                        onClick={() => toggleMut.mutate({ indexerId: idx.id, monitored: !idx.monitored })}
                        disabled={toggleMut.isPending}
                        className={`rounded px-2 py-1 text-xs font-medium transition-colors disabled:opacity-50 ${
                          idx.monitored
                            ? 'text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/30'
                            : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                        }`}
                      >
                        {idx.monitored ? 'Disable' : 'Enable'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}

// ─── Apps page ──────────────────────────────────────────────────────────────

export function Apps() {
  const qc = useQueryClient()

  const { data: instances, isLoading, error } = useQuery({
    queryKey: ['instances'],
    queryFn: api.instances.list,
  })

  // Add dialog
  const [showAdd, setShowAdd] = useState(false)
  const [addForm, setAddForm] = useState<FormState>(defaultForm())

  // Edit dialog
  const [editTarget, setEditTarget] = useState<Instance | null>(null)
  const [editForm, setEditForm] = useState<FormState>(defaultForm())

  // Per-row test-connection results
  const [testResult, setTestResult] = useState<Record<string, { ok: boolean; msg: string }>>({})

  // Jackett indexer modal
  const [jackettTarget, setJackettTarget] = useState<Instance | null>(null)

  // ── helpers ──
  function invalidate() {
    qc.invalidateQueries({ queryKey: ['instances'] })
  }

  // ── mutations ──
  const createMut = useMutation({
    mutationFn: (f: FormState) =>
      api.instances.create({
        kind: f.kind,
        name: f.name,
        base_url: f.base_url,
        api_key: f.api_key || undefined,
      }),
    onSuccess: () => {
      invalidate()
      setShowAdd(false)
      setAddForm(defaultForm())
    },
  })

  const updateMut = useMutation({
    mutationFn: ({ id, f }: { id: string; f: FormState }) =>
      api.instances.update(id, {
        name: f.name,
        base_url: f.base_url,
        api_key: f.api_key || undefined,
      }),
    onSuccess: () => {
      invalidate()
      setEditTarget(null)
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.instances.delete(id),
    onSuccess: invalidate,
  })

  const enableMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.instances.setEnabled(id, enabled),
    onSuccess: invalidate,
  })

  const testMut = useMutation({
    mutationFn: (id: string) => api.instances.test(id),
    onSuccess(data, id) {
      setTestResult(prev => ({
        ...prev,
        [id]: data.ok
          ? { ok: true, msg: 'Connected!' }
          : { ok: false, msg: data.error ?? 'Connection failed' },
      }))
    },
    onError(err: Error, id) {
      setTestResult(prev => ({
        ...prev,
        [id]: { ok: false, msg: err.message },
      }))
    },
  })

  // ── handlers ──
  function openEdit(inst: Instance) {
    setEditTarget(inst)
    setEditForm({ kind: inst.kind, name: inst.name, base_url: inst.base_url, api_key: '' })
  }

  function handleDelete(inst: Instance) {
    if (window.confirm(`Delete "${inst.name}"? This cannot be undone.`)) {
      deleteMut.mutate(inst.id)
    }
  }

  function handleTest(inst: Instance) {
    // Clear previous result before re-testing
    setTestResult(prev => {
      const next = { ...prev }
      delete next[inst.id]
      return next
    })
    testMut.mutate(inst.id)
  }

  // ── render ──
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Apps</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage your monitored applications
          </p>
        </div>
        <button
          onClick={() => { setAddForm(defaultForm()); setShowAdd(true) }}
          className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
          Add App
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
          Failed to load apps: {(error as Error).message}
        </div>
      )}

      {/* Empty state */}
      {instances && instances.length === 0 && (
        <div className="rounded-xl border-2 border-dashed border-gray-200 dark:border-gray-700 py-20 text-center">
          <svg className="mx-auto h-12 w-12 text-gray-300 dark:text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
          </svg>
          <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">No apps yet</p>
          <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">Click "Add App" to get started.</p>
        </div>
      )}

      {/* Table */}
      {instances && instances.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                {['Kind', 'Name', 'Base URL', 'Status', 'Actions'].map(h => (
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
              {instances.map(inst => {
                const tr = testResult[inst.id]
                const isTesting = testMut.isPending && testMut.variables === inst.id
                const isDeleting = deleteMut.isPending && deleteMut.variables === inst.id
                const isToggling = enableMut.isPending && enableMut.variables?.id === inst.id

                return (
                  <tr
                    key={inst.id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                  >
                    {/* Kind */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${KIND_COLORS[inst.kind]}`}>
                        {inst.kind}
                      </span>
                    </td>

                    {/* Name */}
                    <td className="px-4 py-3 whitespace-nowrap font-medium text-gray-900 dark:text-gray-100">
                      {inst.name}
                    </td>

                    {/* Base URL */}
                    <td className="max-w-[220px] truncate px-4 py-3 font-mono text-xs text-gray-500 dark:text-gray-400">
                      {inst.base_url}
                    </td>

                    {/* Status */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          inst.enabled
                            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                            : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${inst.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
                        />
                        {inst.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>

                    {/* Actions */}
                    <td className="px-4 py-3">
                      <div className="flex flex-col gap-1.5">
                        <div className="flex flex-wrap items-center gap-2">
                          {/* Test */}
                          <button
                            onClick={() => handleTest(inst)}
                            disabled={isTesting || isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 disabled:opacity-50 transition-colors"
                          >
                            {isTesting ? 'Testing…' : 'Test'}
                          </button>

                          {/* Jackett indexers */}
                          {inst.kind === 'jackett' && (
                            <button
                              onClick={() => setJackettTarget(inst)}
                              disabled={isDeleting}
                              className="rounded px-2 py-1 text-xs font-medium text-teal-600 dark:text-teal-400 hover:bg-teal-50 dark:hover:bg-teal-900/30 disabled:opacity-50 transition-colors"
                            >
                              Indexers
                            </button>
                          )}

                          {/* Toggle enabled */}
                          <button
                            onClick={() => enableMut.mutate({ id: inst.id, enabled: !inst.enabled })}
                            disabled={isToggling || isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-900/30 disabled:opacity-50 transition-colors"
                          >
                            {isToggling ? '…' : inst.enabled ? 'Disable' : 'Enable'}
                          </button>

                          {/* Edit */}
                          <button
                            onClick={() => openEdit(inst)}
                            disabled={isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
                          >
                            Edit
                          </button>

                          {/* Delete */}
                          <button
                            onClick={() => handleDelete(inst)}
                            disabled={isDeleting}
                            className="rounded px-2 py-1 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 disabled:opacity-50 transition-colors"
                          >
                            {isDeleting ? 'Deleting…' : 'Delete'}
                          </button>
                        </div>

                        {/* Inline test result */}
                        {tr && (
                          <p className={`text-xs ${tr.ok ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                            {tr.msg}
                          </p>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Add dialog ── */}
      {showAdd && (
        <Modal
          title="Add App"
          onClose={() => setShowAdd(false)}
          onSubmit={() => createMut.mutate(addForm)}
          submitLabel="Add"
          submitting={createMut.isPending}
        >
          <AppForm form={addForm} onChange={setAddForm} showKind disabled={createMut.isPending} />
          {createMut.isError && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {(createMut.error as Error).message}
            </p>
          )}
        </Modal>
      )}

      {/* ── Edit dialog ── */}
      {editTarget && (
        <Modal
          title={`Edit — ${editTarget.name}`}
          onClose={() => setEditTarget(null)}
          onSubmit={() => updateMut.mutate({ id: editTarget.id, f: editForm })}
          submitLabel="Save"
          submitting={updateMut.isPending}
        >
          <AppForm form={editForm} onChange={setEditForm} showKind={false} disabled={updateMut.isPending} />
          {updateMut.isError && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {(updateMut.error as Error).message}
            </p>
          )}
        </Modal>
      )}

      {/* ── Jackett indexers modal ── */}
      {jackettTarget && (
        <JackettIndexersModal
          instance={jackettTarget}
          onClose={() => setJackettTarget(null)}
        />
      )}
    </div>
  )
}
