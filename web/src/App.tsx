import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { MigrationDetailPage } from './pages/MigrationDetailPage'
import { MigrationDiffPage } from './pages/MigrationDiffPage'
import { MigrationsPage } from './pages/MigrationsPage'
import { TeamPage } from './pages/TeamPage'

function SettingsPage() {
  return <div className="font-body-md text-body-md text-on-surface">Settings</div>
}

function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<Navigate replace to="/migrations" />} />
        <Route path="migrations" element={<MigrationsPage />} />
        <Route path="migrations/:version" element={<MigrationDetailPage />} />
        <Route path="migrations/:version/diff" element={<MigrationDiffPage />} />
        <Route path="team" element={<TeamPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  )
}

export default App
