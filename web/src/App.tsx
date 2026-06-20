import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'

function MigrationsPage() {
  return (
    <div>
      <div className="mb-unit-8">
        <h2 className="font-display text-display text-on-background">Migration Dashboard</h2>
        <p className="font-body-md text-body-md text-on-surface-variant">
          Manage and track database schema evolution.
        </p>
      </div>
      <div className="rounded border border-outline-variant bg-surface-container p-unit-4 text-on-surface-variant">
        Migration dashboard implementation is next.
      </div>
    </div>
  )
}

function MigrationDetailPage() {
  return <div className="font-body-md text-body-md text-on-surface">Migration detail</div>
}

function MigrationDiffPage() {
  return <div className="font-body-md text-body-md text-on-surface">Schema diff</div>
}

function TeamPage() {
  return <div className="font-body-md text-body-md text-on-surface">Team</div>
}

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
