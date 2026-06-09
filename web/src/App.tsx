// web/src/App.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Layout } from './components/Layout'
import { Dashboard } from './pages/Dashboard'
import { Logs } from './pages/Logs'
import { Issues } from './pages/Issues'
import { Apps } from './pages/Apps'
import { Backup } from './pages/Backup'
import { Sync } from './pages/Sync'
import { Settings } from './pages/Settings'

const queryClient = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="logs" element={<Logs />} />
            <Route path="issues" element={<Issues />} />
            <Route path="apps" element={<Apps />} />
            <Route path="backup" element={<Backup />} />
            <Route path="sync" element={<Sync />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
