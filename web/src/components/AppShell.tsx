import { useEffect } from 'react'
import { Link, Outlet, useLocation } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'

type NavItem = {
  to: string
  icon: string
  label: string
  isActive: (pathname: string) => boolean
}

function buildNavItems(pathname: string): NavItem[] {
  const diffPath = /^\/migrations\/[^/]+\/diff$/.test(pathname) ? pathname : '/migrations'
  return [
    { to: '/', icon: 'dashboard', label: 'Dashboard', isActive: (path) => path === '/' },
    {
      to: '/migrations',
      icon: 'database_off',
      label: 'Migrations',
      isActive: (path) => path === '/migrations' || /^\/migrations\/[^/]+$/.test(path),
    },
    { to: diffPath, icon: 'difference', label: 'Schema Diff', isActive: (path) => /^\/migrations\/[^/]+\/diff$/.test(path) },
    { to: '/team', icon: 'group', label: 'Team', isActive: (path) => path === '/team' },
    { to: '/settings', icon: 'settings', label: 'Settings', isActive: (path) => path === '/settings' },
  ]
}

export function AppShell() {
  const environmentName = useAppStore((state) => state.environmentName)
  const theme = useAppStore((state) => state.theme)
  const toggleTheme = useAppStore((state) => state.toggleTheme)
  const { pathname } = useLocation()
  const navItems = buildNavItems(pathname)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  return (
    <main className="min-h-screen bg-background text-on-surface">
      <aside className="fixed inset-y-0 left-0 hidden w-sidebar-width border-r border-outline-variant bg-surface-container p-unit-4 lg:flex lg:flex-col">
        <div className="mb-unit-8 flex items-center gap-unit-2">
          <div className="flex h-8 w-8 items-center justify-center rounded bg-primary-container text-on-primary-container">
            <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>
              database
            </span>
          </div>
          <div>
            <h1 className="font-display text-headline-sm font-bold text-primary">Rift DB</h1>
            <p className="font-label-caps text-label-caps uppercase text-on-surface-variant">
              {environmentName}
            </p>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-unit-1 font-body-md text-body-md">
          {navItems.map((item) => {
            const isActive = item.isActive(pathname)
            return (
              <Link
                key={`${item.to}-${item.label}`}
                to={item.to}
                className={[
                  'flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 transition-colors',
                  isActive
                    ? 'border-r-2 border-primary bg-primary-container/10 text-primary'
                    : 'text-on-surface-variant hover:bg-surface-variant hover:text-on-surface',
                ].join(' ')}
              >
                <span className="material-symbols-outlined">{item.icon}</span>
                <span> {item.label}</span>
              </Link>
            )
          })}
        </nav>

        <button className="mb-unit-4 flex w-full items-center justify-center gap-unit-2 rounded bg-primary py-2 font-body-md text-body-md font-medium text-on-primary transition-colors hover:bg-primary/90 active:scale-95">
          <span className="material-symbols-outlined text-[18px]">add</span>
          New Migration
        </button>
      </aside>

      <section className="lg:pl-sidebar-width">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-outline-variant bg-background px-gutter">
          <div className="flex gap-unit-4 font-label-caps text-label-caps uppercase text-on-surface-variant">
            <a className="hover:text-on-surface" href="/README.md">
              Docs
            </a>
            <a className="hover:text-on-surface" href="/api/v1/status">
              API
            </a>
            <span>Rift</span>
          </div>
          <div className="flex items-center gap-unit-2">
            <button
              aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
              className="flex items-center gap-unit-2 rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant"
              type="button"
              onClick={toggleTheme}
            >
              <span className="material-symbols-outlined text-[18px]">{theme === 'dark' ? 'light_mode' : 'dark_mode'}</span>
              {theme === 'dark' ? 'Light' : 'Dark'} Mode
            </button>
            <button className="rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant">
              Preview Changes
            </button>
            <button className="rounded bg-primary px-3 py-1.5 font-label-caps text-label-caps font-bold uppercase text-on-primary transition-colors hover:bg-primary/90">
              Apply Migrations
            </button>
          </div>
        </header>

        <div className="mx-auto max-w-container-max p-unit-8">
          <Outlet />
        </div>
      </section>
    </main>
  )
}
