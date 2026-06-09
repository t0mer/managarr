// web/src/pages/Settings.tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { NotifyChannel, NotifyProvider } from '../lib/types'

// ─── Constants ──────────────────────────────────────────────────────────────

const PROVIDER_LABELS: Record<NotifyProvider, string> = {
  shoutrrr: 'Shoutrrr',
  greenapi: 'GreenAPI',
  whatsapp_web: 'WhatsApp Web',
}

const PROVIDER_COLORS: Record<NotifyProvider, string> = {
  shoutrrr:
    'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200',
  greenapi:
    'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  whatsapp_web:
    'bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200',
}

const PROVIDERS: NotifyProvider[] = ['shoutrrr', 'greenapi', 'whatsapp_web']

// ─── Form state ──────────────────────────────────────────────────────────────

interface ShoutrrrFields {
  url: string
}

interface GreenApiFields {
  instance_id: string
  token: string
  phone: string
  api_url: string
}

interface WhatsappWebFields {
  base_url: string
  phone: string
  username: string
  password: string
}

interface ChannelForm {
  name: string
  provider: NotifyProvider
  notify_on_success: boolean
  notify_on_failure: boolean
  enabled: boolean
  shoutrrr: ShoutrrrFields
  greenapi: GreenApiFields
  whatsapp_web: WhatsappWebFields
}

function defaultForm(): ChannelForm {
  return {
    name: '',
    provider: 'shoutrrr',
    notify_on_success: true,
    notify_on_failure: true,
    enabled: true,
    shoutrrr: { url: '' },
    greenapi: { instance_id: '', token: '', phone: '', api_url: '' },
    whatsapp_web: { base_url: '', phone: '', username: '', password: '' },
  }
}

function formToApiBody(form: ChannelForm) {
  let config: Record<string, string> = {}
  if (form.provider === 'shoutrrr') {
    config = { url: form.shoutrrr.url }
  } else if (form.provider === 'greenapi') {
    config = {
      instance_id: form.greenapi.instance_id.trim(),
      token: form.greenapi.token.trim(),
      phone: form.greenapi.phone.trim(),
    }
    if (form.greenapi.api_url.trim()) {
      config.api_url = form.greenapi.api_url.trim()
    }
  } else if (form.provider === 'whatsapp_web') {
    config = {
      base_url: form.whatsapp_web.base_url,
      phone: form.whatsapp_web.phone.trim(),
    }
    if (form.whatsapp_web.username.trim()) {
      config.username = form.whatsapp_web.username.trim()
    }
    if (form.whatsapp_web.password) {
      config.password = form.whatsapp_web.password
    }
  }
  return {
    name: form.name,
    provider: form.provider,
    config,
    notify_on_success: form.notify_on_success,
    notify_on_failure: form.notify_on_failure,
    enabled: form.enabled,
  }
}

// ─── Shared classes ──────────────────────────────────────────────────────────

const inputCls =
  'w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 ' +
  'px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 ' +
  'focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50'

const labelCls = 'text-sm font-medium text-gray-700 dark:text-gray-300'

// ─── Toggle ──────────────────────────────────────────────────────────────────

interface ToggleProps {
  checked: boolean
  onChange: (v: boolean) => void
  label: string
  disabled?: boolean
}

function Toggle({ checked, onChange, label, disabled }: ToggleProps) {
  return (
    <label className="flex cursor-pointer items-center gap-3">
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-50 ${
          checked ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-700'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ${
            checked ? 'translate-x-4' : 'translate-x-0'
          }`}
        />
      </button>
      <span className={labelCls}>{label}</span>
    </label>
  )
}

// ─── Provider-specific fields ────────────────────────────────────────────────

interface ProviderFieldsProps {
  form: ChannelForm
  onChange: (f: ChannelForm) => void
  disabled?: boolean
}

function ShoutrrrFieldSet({ form, onChange, disabled }: ProviderFieldsProps) {
  return (
    <div className="flex flex-col gap-1">
      <label className={labelCls}>URL</label>
      <input
        className={inputCls}
        type="text"
        placeholder="slack://token@channel"
        value={form.shoutrrr.url}
        disabled={disabled}
        onChange={e =>
          onChange({ ...form, shoutrrr: { url: e.target.value } })
        }
        autoComplete="off"
      />
      <p className="text-xs text-gray-400">
        Shoutrrr URL — supports Slack, Discord, Telegram, Gotify, SMTP, ntfy,
        and more.
      </p>
    </div>
  )
}

function GreenApiFieldSet({ form, onChange, disabled }: ProviderFieldsProps) {
  const g = form.greenapi
  const set = (patch: Partial<GreenApiFields>) =>
    onChange({ ...form, greenapi: { ...g, ...patch } })
  return (
    <>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>Instance ID</label>
        <input
          className={inputCls}
          type="text"
          placeholder="1101234567"
          value={g.instance_id}
          disabled={disabled}
          onChange={e => set({ instance_id: e.target.value })}
          autoComplete="off"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>Token</label>
        <input
          className={inputCls}
          type="password"
          placeholder="leave blank to keep existing"
          value={g.token}
          disabled={disabled}
          onChange={e => set({ token: e.target.value })}
          autoComplete="new-password"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>Recipient Phone</label>
        <input
          className={inputCls}
          type="text"
          placeholder="972501234567 (digits only, no + or spaces)"
          value={g.phone}
          disabled={disabled}
          onChange={e => set({ phone: e.target.value })}
          autoComplete="off"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>
          API URL{' '}
          <span className="font-normal text-gray-400">(optional)</span>
        </label>
        <input
          className={inputCls}
          type="url"
          placeholder="https://api.green-api.com"
          value={g.api_url}
          disabled={disabled}
          onChange={e => set({ api_url: e.target.value })}
          autoComplete="off"
        />
      </div>
    </>
  )
}

function WhatsappWebFieldSet({ form, onChange, disabled }: ProviderFieldsProps) {
  const w = form.whatsapp_web
  const set = (patch: Partial<WhatsappWebFields>) =>
    onChange({ ...form, whatsapp_web: { ...w, ...patch } })
  return (
    <>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>Base URL</label>
        <input
          className={inputCls}
          type="url"
          placeholder="http://localhost:3000"
          value={w.base_url}
          disabled={disabled}
          onChange={e => set({ base_url: e.target.value })}
          autoComplete="off"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>Recipient Phone</label>
        <input
          className={inputCls}
          type="text"
          placeholder="972501234567 (digits only, no + or spaces)"
          value={w.phone}
          disabled={disabled}
          onChange={e => set({ phone: e.target.value })}
          autoComplete="off"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>
          Username{' '}
          <span className="font-normal text-gray-400">(optional)</span>
        </label>
        <input
          className={inputCls}
          type="text"
          placeholder="admin"
          value={w.username}
          disabled={disabled}
          onChange={e => set({ username: e.target.value })}
          autoComplete="off"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className={labelCls}>
          Password{' '}
          <span className="font-normal text-gray-400">(optional)</span>
        </label>
        <input
          className={inputCls}
          type="password"
          placeholder="leave blank to keep existing"
          value={w.password}
          disabled={disabled}
          onChange={e => set({ password: e.target.value })}
          autoComplete="new-password"
        />
      </div>
    </>
  )
}

// ─── Channel form ────────────────────────────────────────────────────────────

interface ChannelFormProps {
  form: ChannelForm
  onChange: (f: ChannelForm) => void
  disabled?: boolean
  showProviderSelect: boolean
}

function ChannelFormFields({
  form,
  onChange,
  disabled,
  showProviderSelect,
}: ChannelFormProps) {
  return (
    <div className="flex flex-col gap-4">
      {showProviderSelect && (
        <div className="flex flex-col gap-1">
          <label className={labelCls}>Provider</label>
          <select
            className={inputCls}
            value={form.provider}
            disabled={disabled}
            onChange={e =>
              onChange({ ...form, provider: e.target.value as NotifyProvider })
            }
          >
            {PROVIDERS.map(p => (
              <option key={p} value={p}>
                {PROVIDER_LABELS[p]}
              </option>
            ))}
          </select>
        </div>
      )}

      <div className="flex flex-col gap-1">
        <label className={labelCls}>Name</label>
        <input
          className={inputCls}
          type="text"
          placeholder="My Slack channel"
          value={form.name}
          disabled={disabled}
          onChange={e => onChange({ ...form, name: e.target.value })}
        />
      </div>

      {/* Provider-specific fields */}
      {form.provider === 'shoutrrr' && (
        <ShoutrrrFieldSet form={form} onChange={onChange} disabled={disabled} />
      )}
      {form.provider === 'greenapi' && (
        <GreenApiFieldSet form={form} onChange={onChange} disabled={disabled} />
      )}
      {form.provider === 'whatsapp_web' && (
        <WhatsappWebFieldSet form={form} onChange={onChange} disabled={disabled} />
      )}

      {/* Toggles */}
      <div className="flex flex-col gap-3 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        <Toggle
          checked={form.notify_on_success}
          onChange={v => onChange({ ...form, notify_on_success: v })}
          label="Notify on success"
          disabled={disabled}
        />
        <Toggle
          checked={form.notify_on_failure}
          onChange={v => onChange({ ...form, notify_on_failure: v })}
          label="Notify on failure / partial failure"
          disabled={disabled}
        />
        <Toggle
          checked={form.enabled}
          onChange={v => onChange({ ...form, enabled: v })}
          label="Enabled"
          disabled={disabled}
        />
      </div>
    </div>
  )
}

// ─── Modal ───────────────────────────────────────────────────────────────────

interface ModalProps {
  title: string
  onClose: () => void
  onSubmit: () => void
  submitLabel: string
  submitting?: boolean
  children: React.ReactNode
  extraFooter?: React.ReactNode
}

function Modal({
  title,
  onClose,
  onSubmit,
  submitLabel,
  submitting,
  children,
  extraFooter,
}: ModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={e => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="w-full max-w-lg rounded-xl bg-white dark:bg-gray-900 shadow-2xl flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4 shrink-0">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {title}
          </h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            aria-label="Close"
          >
            <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="overflow-y-auto px-6 py-5 flex-1">{children}</div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-gray-200 dark:border-gray-700 px-6 py-4 shrink-0">
          <div>{extraFooter}</div>
          <div className="flex gap-3">
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
    </div>
  )
}

// ─── Indicator dot ───────────────────────────────────────────────────────────

function Dot({ active, title }: { active: boolean; title: string }) {
  return (
    <span
      title={title}
      className={`inline-block h-2 w-2 rounded-full ${
        active ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
      }`}
    />
  )
}

// ─── Settings page ───────────────────────────────────────────────────────────

export function Settings() {
  const qc = useQueryClient()

  const {
    data: channels,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['notify-channels'],
    queryFn: api.notify.list,
  })

  // Add dialog
  const [showAdd, setShowAdd] = useState(false)
  const [addForm, setAddForm] = useState<ChannelForm>(defaultForm())
  const [addTestResult, setAddTestResult] = useState<{
    ok: boolean
    msg: string
  } | null>(null)

  // Edit dialog
  const [editTarget, setEditTarget] = useState<NotifyChannel | null>(null)
  const [editForm, setEditForm] = useState<ChannelForm>(defaultForm())
  const [editTestResult, setEditTestResult] = useState<{
    ok: boolean
    msg: string
  } | null>(null)

  // ── helpers ──
  function invalidate() {
    qc.invalidateQueries({ queryKey: ['notify-channels'] })
  }

  function openEdit(ch: NotifyChannel) {
    const form = defaultForm()
    form.name = ch.name
    form.provider = ch.provider
    form.notify_on_success = ch.notify_on_success
    form.notify_on_failure = ch.notify_on_failure
    form.enabled = ch.enabled
    // credentials left blank — backend never returns them
    setEditTarget(ch)
    setEditForm(form)
    setEditTestResult(null)
  }

  // ── mutations ──
  const createMut = useMutation({
    mutationFn: (f: ChannelForm) => api.notify.create(formToApiBody(f)),
    onSuccess: () => {
      invalidate()
      setShowAdd(false)
      setAddForm(defaultForm())
      setAddTestResult(null)
    },
  })

  const updateMut = useMutation({
    mutationFn: ({ id, f }: { id: string; f: ChannelForm }) =>
      api.notify.update(id, formToApiBody(f)),
    onSuccess: () => {
      invalidate()
      setEditTarget(null)
      setEditTestResult(null)
    },
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.notify.delete(id),
    onSuccess: invalidate,
  })

  const addTestMut = useMutation({
    mutationFn: (f: ChannelForm) =>
      api.notify.test(formToApiBody(f)),
    onSuccess(data) {
      setAddTestResult(
        data.ok
          ? { ok: true, msg: 'Test message sent!' }
          : { ok: false, msg: data.error ?? 'Test failed' }
      )
    },
    onError(err: Error) {
      setAddTestResult({ ok: false, msg: err.message })
    },
  })

  const editTestMut = useMutation({
    mutationFn: (f: ChannelForm) =>
      api.notify.test(formToApiBody(f)),
    onSuccess(data) {
      setEditTestResult(
        data.ok
          ? { ok: true, msg: 'Test message sent!' }
          : { ok: false, msg: data.error ?? 'Test failed' }
      )
    },
    onError(err: Error) {
      setEditTestResult({ ok: false, msg: err.message })
    },
  })

  function handleDelete(ch: NotifyChannel) {
    if (window.confirm(`Delete channel "${ch.name}"? This cannot be undone.`)) {
      deleteMut.mutate(ch.id)
    }
  }

  // ── render ──
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Settings
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage notification channels
          </p>
        </div>
        <button
          onClick={() => {
            setAddForm(defaultForm())
            setAddTestResult(null)
            setShowAdd(true)
          }}
          className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
        >
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path
              fillRule="evenodd"
              d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
              clipRule="evenodd"
            />
          </svg>
          Add Channel
        </button>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex items-center justify-center py-20 text-gray-500 dark:text-gray-400">
          <svg
            className="mr-3 h-5 w-5 animate-spin"
            viewBox="0 0 24 24"
            fill="none"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8v8H4z"
            />
          </svg>
          Loading…
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load channels: {(error as Error).message}
        </div>
      )}

      {/* Empty state */}
      {channels && channels.length === 0 && (
        <div className="rounded-xl border-2 border-dashed border-gray-200 dark:border-gray-700 py-20 text-center">
          <svg
            className="mx-auto h-12 w-12 text-gray-300 dark:text-gray-600"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0"
            />
          </svg>
          <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">
            No notification channels yet
          </p>
          <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">
            Click "Add Channel" to configure your first notification channel.
          </p>
        </div>
      )}

      {/* Channel list */}
      {channels && channels.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                {['Name', 'Provider', 'Notifications', 'Status', 'Actions'].map(
                  h => (
                    <th
                      key={h}
                      className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
                    >
                      {h}
                    </th>
                  )
                )}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-800 bg-white dark:bg-gray-900">
              {channels.map(ch => {
                const isDeleting =
                  deleteMut.isPending && deleteMut.variables === ch.id

                return (
                  <tr
                    key={ch.id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                  >
                    {/* Name */}
                    <td className="px-4 py-3 whitespace-nowrap font-medium text-gray-900 dark:text-gray-100">
                      {ch.name}
                    </td>

                    {/* Provider */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${PROVIDER_COLORS[ch.provider]}`}
                      >
                        {PROVIDER_LABELS[ch.provider]}
                      </span>
                    </td>

                    {/* Notification indicators */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex items-center gap-3">
                        <span className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
                          <Dot
                            active={ch.notify_on_success}
                            title={
                              ch.notify_on_success
                                ? 'Notifies on success'
                                : 'Success notifications off'
                            }
                          />
                          Success
                        </span>
                        <span className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
                          <Dot
                            active={ch.notify_on_failure}
                            title={
                              ch.notify_on_failure
                                ? 'Notifies on failure'
                                : 'Failure notifications off'
                            }
                          />
                          Failure
                        </span>
                      </div>
                    </td>

                    {/* Enabled status */}
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          ch.enabled
                            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                            : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${
                            ch.enabled ? 'bg-green-500' : 'bg-gray-400'
                          }`}
                        />
                        {ch.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>

                    {/* Actions */}
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => openEdit(ch)}
                          disabled={isDeleting}
                          className="rounded px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDelete(ch)}
                          disabled={isDeleting}
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
          title="Add Notification Channel"
          onClose={() => {
            setShowAdd(false)
            setAddTestResult(null)
          }}
          onSubmit={() => createMut.mutate(addForm)}
          submitLabel="Add"
          submitting={createMut.isPending}
          extraFooter={
            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => {
                  setAddTestResult(null)
                  addTestMut.mutate(addForm)
                }}
                disabled={addTestMut.isPending || createMut.isPending}
                className="rounded-md border border-gray-300 dark:border-gray-600 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
              >
                {addTestMut.isPending ? 'Sending…' : 'Send Test'}
              </button>
              {addTestResult && (
                <p
                  className={`text-xs ${
                    addTestResult.ok
                      ? 'text-green-600 dark:text-green-400'
                      : 'text-red-600 dark:text-red-400'
                  }`}
                >
                  {addTestResult.msg}
                </p>
              )}
            </div>
          }
        >
          <ChannelFormFields
            form={addForm}
            onChange={f => {
              setAddForm(f)
              setAddTestResult(null)
            }}
            showProviderSelect
            disabled={createMut.isPending}
          />
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
          onClose={() => {
            setEditTarget(null)
            setEditTestResult(null)
          }}
          onSubmit={() => updateMut.mutate({ id: editTarget.id, f: editForm })}
          submitLabel="Save"
          submitting={updateMut.isPending}
          extraFooter={
            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => {
                  setEditTestResult(null)
                  editTestMut.mutate(editForm)
                }}
                disabled={editTestMut.isPending || updateMut.isPending}
                className="rounded-md border border-gray-300 dark:border-gray-600 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
              >
                {editTestMut.isPending ? 'Sending…' : 'Send Test'}
              </button>
              {editTestResult && (
                <p
                  className={`text-xs ${
                    editTestResult.ok
                      ? 'text-green-600 dark:text-green-400'
                      : 'text-red-600 dark:text-red-400'
                  }`}
                >
                  {editTestResult.msg}
                </p>
              )}
            </div>
          }
        >
          <ChannelFormFields
            form={editForm}
            onChange={f => {
              setEditForm(f)
              setEditTestResult(null)
            }}
            showProviderSelect={false}
            disabled={updateMut.isPending}
          />
          {editTarget.provider !== editForm.provider && (
            <p className="mt-2 text-xs text-gray-400">
              Provider: {PROVIDER_LABELS[editTarget.provider]} — fields cleared
              since credentials are not returned by the API.
            </p>
          )}
          {updateMut.isError && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {(updateMut.error as Error).message}
            </p>
          )}
        </Modal>
      )}
    </div>
  )
}
