// web/src/pages/Dashboard.tsx
function ComingSoon({ name }: { name: string }) {
  return (
    <div className="flex items-center justify-center h-full min-h-[60vh]">
      <div className="text-center p-10 rounded-xl border border-[var(--border)] bg-[var(--sidebar-bg)] max-w-sm w-full">
        <h1 className="text-2xl font-semibold mb-2">{name}</h1>
        <p className="text-sm opacity-60">Coming soon</p>
      </div>
    </div>
  )
}
export function Dashboard() { return <ComingSoon name="Dashboard" /> }
