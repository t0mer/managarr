// web/src/pages/Sync.tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { SyncJob, SyncPreview, Instance, ProviderKind } from '../lib/types'

// ─── Helpers ────────────────────────────────────────────────────────────────

function instanceName(instances: Instance[] | undefined, id: string): string {
  return instances?.find(i => i.id === id)?.name ?? id
}

function instanceKind(instances: Instance[] | undefined, id: string): ProviderKind | undefined {
  return instances?.find(i => i.id === id)?.kind
}

// ─── Toast ──────────────────────────────────────────────────────────────────

interface Toast {
  id: number
  message: string
  ok: boolean
}

let toastCounter = 0

function useToasts() {
  const [toasts, setToasts] = useState<Toast[]>([])

  function push(message: string, ok: boolean) {
    const id = ++toastCounter
    setToasts(prev => [...prev, { id, message, ok }])
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 4000)
  }

  return { toasts, push }
}

function ToastContainer({ toasts }: { toasts: Toast[] }) {
  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-2 pointer-events-none">
      {toasts.map(t => (
        <div
          key={t.id}
          className={`rounded-lg px-4 py-3 text-sm font-medium text-white shadow-lg transition-all ${
            t.ok ? 'bg-green-600' : 'bg-red-600'
          }`}
        >
          {t.message}
        </div>
      ))}
    </div>
  )
}

// ─── Modal ──────────────────────────────────────────────────────────────────

interface ModalProps {
  title: string
  onClose: () => void
  onSubmit?: () => void
  submitLabel?: string
  submitting?: boolean
  children: React.ReactNode
  wide?: boolean
}

function Modal({ title, onClose, onSubmit, submitLabel, submitting, children, wide }: ModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className={`w-full ${wide ? 'max-w-2xl' : 'max-w-md'} rounded-xl bg-white dark:bg-gray-900 shadow-2xl`}>
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
        {onSubmit && (
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
              {submitting ? 'Saving…' : submitLabel ?? 'Save'}
            </button>
          </div>
        )}
        {!onSubmit && (
          <div className="flex justify-end border-t border-gray-200 dark:border-gray-700 px-6 py-4">
            <button
              onClick={onClose}
              className="rounded-md px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              Close
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Add Job Form ────────────────────────────────────────────────────────────

interface AddJobFormProps {
  instances: Instance[]
  sourceId: string
  targetId: string
  schedule: string
  enabled: boolean
  onChange: (patch: Partial<{ sourceId: string; targetId: string; schedule: string; enabled: boolean }>) => void
  disabled?: boolean
}

function AddJobForm({ instances, sourceId, targetId, schedule, enabled, onChange, disabled }: AddJobFormProps) {
  const inputCls =
    'w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 ' +
    'px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 ' +
    'focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50'

  // Filter target instances: must be same kind as source, and not the same instance
  const sourceKind = instances.find(i => i.id === sourceId)?.kind
  const targetCandidates = sourceKind
    ? instances.filter(i => i.kind === sourceKind && i.id !== sourceId)
    : instances.filter(i => i.id !== sourceId)

  // If current targetId is no longer valid after source change, we show it but it's a mismatch
  const targetValid = targetCandidates.some(i => i.id === targetId)

  return (
    <div className="flex flex-col gap-4">
      {/* Source */}
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Source Instance</label>
        <select
          className={inputCls}
          value={sourceId}
          disabled={disabled}
          onChange={e => {
            const newSource = e.target.value
            // Reset target if it's no longer a valid match
            const newKind = instances.find(i => i.id === newSource)?.kind
            const newTarget = targetId && instances.find(i => i.id === targetId)?.kind === newKind && targetId !== newSource
              ? targetId
              : ''
            onChange({ sourceId: newSource, targetId: newTarget })
          }}
        >
          <option value="">— select source —</option>
          {instances.map(i => (
            <option key={i.id} value={i.id}>
              {i.name} ({i.kind})
            </option>
          ))}
        </select>
      </div>

      {/* Target */}
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Target Instance
          {sourceKind && (
            <span className="ml-1 text-xs font-normal text-gray-400">(same kind: {sourceKind})</span>
          )}
        </label>
        <select
          className={inputCls}
          value={targetValid ? targetId : ''}
          disabled={disabled || !sourceId}
          onChange={e => onChange({ targetId: e.target.value })}
        >
          <option value="">— select target —</option>
          {targetCandidates.map(i => (
            <option key={i.id} value={i.id}>
              {i.name} ({i.kind})
            </option>
          ))}
        </select>
        {sourceId && targetCandidates.length === 0 && (
          <p className="text-xs text-amber-600 dark:text-amber-400">
            No other {sourceKind} instances available to sync to.
          </p>
        )}
      </div>

      {/* Schedule */}
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Schedule <span className="font-normal text-gray-400">(optional cron)</span>
        </label>
        <input
          className={inputCls}
          type="text"
          placeholder="e.g. 0 * * * * (every hour)"
          value={schedule}
          disabled={disabled}
          onChange={e => onChange({ schedule: e.target.value })}
        />
      </div>

      {/* Enabled */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          disabled={disabled}
          onClick={() => onChange({ enabled: !enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-1 disabled:opacity-50 ${
            enabled ? 'bg-indigo-600' : 'bg-gray-300 dark:bg-gray-600'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
              enabled ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
        <span className="text-sm text-gray-700 dark:text-gray-300">Enabled</span>
      </div>
    </div>
  )
}

// ─── Preview Panel ───────────────────────────────────────────────────────────

function PreviewPanel({ preview }: { preview: SyncPreview }) {
  if (preview.count === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-10 text-center">
        <svg className="h-10 w-10 text-green-400 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p className="text-base font-medium text-gray-700 dark:text-gray-300">Already in sync</p>
        <p className="text-sm text-gray-400 dark:text-gray-500 mt-1">No changes would be applied.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {preview.count} change{preview.count !== 1 ? 's' : ''} would be applied:
      </p>
      <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800">
            <tr>
              {['Field', 'Current value', 'New value'].map(h => (
                <th
                  key={h}
                  className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 dark:divide-gray-800 bg-white dark:bg-gray-900">
            {preview.changes.map((c, idx) => (
              <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                <td className="px-3 py-2 font-mono text-xs text-gray-700 dark:text-gray-300 whitespace-nowrap">
                  {c.field}
                </td>
                <td className="px-3 py-2 max-w-[180px] truncate text-xs text-red-600 dark:text-red-400 font-mono">
                  {c.old_value === null || c.old_value === undefined
                    ? <span className="italic text-gray-400">—</span>
                    : String(c.old_value)}
                </td>
                <td className="px-3 py-2 max-w-[180px] truncate text-xs text-green-600 dark:text-green-400 font-mono">
                  {c.new_value === null || c.new_value === undefined
                    ? <span className="italic text-gray-400">—</span>
                    : String(c.new_value)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ─── Sync Page ───────────────────────────────────────────────────────────────

interface AddFormState {
  sourceId: string
  targetId: string
  schedule: string
  enabled: boolean
}

function defaultAddForm(): AddFormState {
  return { sourceId: '', targetId: '', schedule: '', enabled: true }
}

export function Sync() {
  const qc = useQueryClient()
  const { toasts, push } = useToasts()

  // ── Queries ──
  const { data: jobs, isLoading: jobsLoading, error: jobsError } = useQuery({
    queryKey: ['sync-jobs'],
    queryFn: api.sync.listJobs,
  })

  const { data: instances, isLoading: instancesLoading } = useQuery({
    queryKey: ['instances'],
    queryFn: api.instances.list,
  })

  // ── Dialog state ──
  const [showAdd, setShowAdd] = useState(false)
  const [addForm, setAddForm] = useState<AddFormState>(defaultAddForm())

  // ── Preview state: jobId → SyncPreview | 'loading' | Error ──
  const [previewJob, setPreviewJob] = useState<SyncJob | null>(null)
  const [previewData, setPreviewData] = useState<SyncPreview | null>(null)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  // ── Per-row apply loading ──
  const [applyingId, setApplyingId] = useState<string | null>(null)

  // ── Mutations ──
  function invalidate() {
    qc.invalidateQueries({ queryKey: ['sync-jobs'] })
  }

  const createMut = useMutation({
    mutationFn: (f: AddFormState) =>
      api.sync.createJob({
        source_instance_id: f.sourceId,
        target_instance_id: f.targetId,
        schedule: f.schedule || undefined,
        enabled: f.enabled,
      }),
    onSuccess: () => {
      invalidate()
      setShowAdd(false)
      setAddForm(defaultAddForm())
      push('Sync job created.', true)
    },
    onError: (err: Error) => {
      push(`Failed to create job: ${err.message}`, false)
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.sync.deleteJob(id),
    onSuccess: () => {
      invalidate()
      push('Sync job deleted.', true)
    },
    onError: (err: Error) => {
      push(`Failed to delete job: ${err.message}`, false)
    },
  })

  // ── Handlers ──
  async function handlePreview(job: SyncJob) {
    setPreviewJob(job)
    setPreviewData(null)
    setPreviewError(null)
    setPreviewLoading(true)
    try {
      const result = await api.sync.preview(job.id)
      setPreviewData(result)
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : String(err))
    } finally {
      setPreviewLoading(false)
    }
  }

  async function handleApply(job: SyncJob) {
    setApplyingId(job.id)
    try {
      const result = await api.sync.apply(job.id)
      push(`Applied ${result.applied} change${result.applied !== 1 ? 's' : ''}.`, true)
    } catch (err) {
      push(`Apply failed: ${err instanceof Error ? err.message : String(err)}`, false)
    } finally {
      setApplyingId(null)
    }
  }

  function handleDelete(job: SyncJob) {
    const src = instanceName(instances, job.source_instance_id)
    const tgt = instanceName(instances, job.target_instance_id)
    if (window.confirm(`Delete sync job "${src} → ${tgt}"? This cannot be undone.`)) {
      deleteMut.mutate(job.id)
    }
  }

  const isLoading = jobsLoading || instancesLoading

  // ── Render ──
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Sync</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Synchronise configuration between instances of the same kind
          </p>
        </div>
        <button
          onClick={() => { setAddForm(defaultAddForm()); setShowAdd(true) }}
          className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
          Add Sync Job
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
      {jobsError && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load sync jobs: {(jobsError as Error).message}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && jobs && jobs.length === 0 && (
        <div className="rounded-xl border-2 border-dashed border-gray-200 dark:border-gray-700 py-20 text-center">
          <svg className="mx-auto h-12 w-12 text-gray-300 dark:text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
          </svg>
          <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">No sync jobs yet</p>
          <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">Click "Add Sync Job" to get started.</p>
        </div>
      )}

      {/* Table */}
      {!isLoading && jobs && jobs.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                {['Source', '', 'Target', 'Schedule', 'Status', 'Actions'].map((h, i) => (
                  <th
                    key={i}
                    className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-800 bg-white dark:bg-gray-900">
              {jobs.map(job => {
                const srcName = instanceName(instances, job.source_instance_id)
                const tgtName = instanceName(instances, job.target_instance_id)
                const srcKind = instanceKind(instances, job.source_instance_id)
                const isDeleting = deleteMut.isPending && deleteMut.variables === job.id
                const isApplying = applyingId === job.id

                return (
                  <tr
                    key={job.id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                  >
                    {/* Source */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-gray-900 dark:text-gray-100">{srcName}</span>
                        {srcKind && (
                          <span className="text-xs text-gray-400 dark:text-gray-500">{srcKind}</span>
                        )}
                      </div>
                    </td>

                    {/* Arrow */}
                    <td className="px-2 py-3 text-gray-400 dark:text-gray-500">
                      <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd" />
                      </svg>
                    </td>

                    {/* Target */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-gray-900 dark:text-gray-100">{tgtName}</span>
                        {srcKind && (
                          <span className="text-xs text-gray-400 dark:text-gray-500">{srcKind}</span>
                        )}
                      </div>
                    </td>

                    {/* Schedule */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      {job.schedule ? (
                        <span className="font-mono text-xs text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 rounded px-2 py-0.5">
                          {job.schedule}
                        </span>
                      ) : (
                        <span className="text-xs text-gray-400 dark:text-gray-500 italic">manual</span>
                      )}
                    </td>

                    {/* Status */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          job.enabled
                            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                            : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${job.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
                        />
                        {job.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>

                    {/* Actions */}
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap items-center gap-2">
                        {/* Preview */}
                        <button
                          onClick={() => handlePreview(job)}
                          disabled={isDeleting || isApplying}
                          className="rounded px-2 py-1 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 disabled:opacity-50 transition-colors"
                        >
                          Preview
                        </button>

                        {/* Apply */}
                        <button
                          onClick={() => handleApply(job)}
                          disabled={isDeleting || isApplying}
                          className="rounded px-2 py-1 text-xs font-medium text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/30 disabled:opacity-50 transition-colors"
                        >
                          {isApplying ? 'Applying…' : 'Apply'}
                        </button>

                        {/* Delete */}
                        <button
                          onClick={() => handleDelete(job)}
                          disabled={isDeleting || isApplying}
                          className="rounded px-2 py-1 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 disabled:opacity-50 transition-colors"
                        >
                          {isDeleting ? 'Deleting…' : 'Delete'}
                        </button>
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
          title="Add Sync Job"
          onClose={() => setShowAdd(false)}
          onSubmit={() => {
            if (!addForm.sourceId || !addForm.targetId) return
            createMut.mutate(addForm)
          }}
          submitLabel="Create"
          submitting={createMut.isPending}
        >
          {instances ? (
            <AddJobForm
              instances={instances}
              sourceId={addForm.sourceId}
              targetId={addForm.targetId}
              schedule={addForm.schedule}
              enabled={addForm.enabled}
              onChange={patch => setAddForm(prev => ({ ...prev, ...patch }))}
              disabled={createMut.isPending}
            />
          ) : (
            <p className="text-sm text-gray-500">Loading instances…</p>
          )}
          {(!addForm.sourceId || !addForm.targetId) && (
            <p className="mt-3 text-xs text-amber-600 dark:text-amber-400">
              Select both a source and a target instance to continue.
            </p>
          )}
          {createMut.isError && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {(createMut.error as Error).message}
            </p>
          )}
        </Modal>
      )}

      {/* ── Preview dialog ── */}
      {previewJob && (
        <Modal
          title={`Preview — ${instanceName(instances, previewJob.source_instance_id)} → ${instanceName(instances, previewJob.target_instance_id)}`}
          onClose={() => { setPreviewJob(null); setPreviewData(null); setPreviewError(null) }}
          wide
        >
          {previewLoading && (
            <div className="flex items-center justify-center py-10 text-gray-500 dark:text-gray-400">
              <svg className="mr-3 h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              Computing diff…
            </div>
          )}
          {previewError && (
            <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-sm text-red-700 dark:text-red-300">
              {previewError}
            </div>
          )}
          {previewData && <PreviewPanel preview={previewData} />}
        </Modal>
      )}

      {/* Toasts */}
      <ToastContainer toasts={toasts} />
    </div>
  )
}
