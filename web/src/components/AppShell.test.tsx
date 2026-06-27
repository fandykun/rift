import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMigration, fetchMigrations } from '../lib/api'
import { useAppStore } from '../stores/appStore'
import { AppShell } from './AppShell'

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    createMigration: vi.fn(),
    fetchMigrations: vi.fn(),
  }
})

function DiffRouteProbe() {
  const location = useLocation()
  return <div>Schema diff content {location.search}</div>
}

function renderShell(initialPath = '/migrations') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route element={<AppShell />} path="/">
            <Route element={<div>Dashboard content</div>} path="migrations" />
            <Route element={<div>Migration detail</div>} path="migrations/:version" />
            <Route element={<DiffRouteProbe />} path="migrations/:version/diff" />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AppShell', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false, theme: 'dark' })
    vi.mocked(fetchMigrations).mockResolvedValue([
      {
        version: '20260623_090000',
        filename: '20260623_090000_create_demo_customers.up.sql',
        status: 'applied',
        applied_by: 'demo',
        has_lint: false,
      },
      {
        version: '20260623_090500',
        filename: '20260623_090500_create_demo_projects.up.sql',
        status: 'pending',
        applied_by: 'demo',
        has_lint: false,
      },
    ])
    vi.mocked(createMigration).mockResolvedValue({
      version: '20260626_221500',
      filename: '20260626_221500_add_billing_events',
      status: 'pending',
      has_lint: false,
    })
  })

  afterEach(() => {
    cleanup()
    document.documentElement.classList.remove('dark')
    vi.clearAllMocks()
  })

  it('toggles the persisted color theme on the document root', () => {
    renderShell()

    expect(screen.getByText('Dashboard content')).not.toBeNull()
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    fireEvent.click(screen.getByRole('button', { name: 'Switch to light mode' }))
    expect(useAppStore.getState().theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)

    fireEvent.click(screen.getByRole('button', { name: 'Switch to dark mode' }))
    expect(useAppStore.getState().theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('links the schema diff menu to a real migration diff route', async () => {
    renderShell()

    await waitFor(() => {
      expect(screen.getByRole('link', { name: /schema diff/i }).getAttribute('href')).toBe('/migrations/20260623_090500/diff')
    })
  })

  it('keeps the schema diff menu scoped to the current migration detail route', () => {
    renderShell('/migrations/20260623_090000')

    expect(screen.getByRole('link', { name: /schema diff/i }).getAttribute('href')).toBe('/migrations/20260623_090000/diff')
  })

  it('creates a migration from the sidebar button and navigates to the detail page', async () => {
    renderShell()

    fireEvent.click(screen.getByRole('button', { name: /new migration/i }))
    fireEvent.change(screen.getByRole('textbox', { name: /migration name/i }), { target: { value: 'Add Billing Events' } })
    expect(screen.getByText('YYYYMMDD_HHMMSS_add_billing_events.up.sql')).not.toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(createMigration).toHaveBeenCalledWith({ name: 'Add Billing Events' }, { token: 'token' })
    })
    expect(await screen.findByText('Migration detail')).not.toBeNull()
  })

  it('navigates header preview and apply buttons to the pending migration diff workflow', async () => {
    renderShell()

    await waitFor(() => {
      expect(screen.getByRole('link', { name: /schema diff/i }).getAttribute('href')).toBe('/migrations/20260623_090500/diff')
    })

    fireEvent.click(screen.getByRole('button', { name: /preview changes/i }))
    expect(await screen.findByText('Schema diff content')).not.toBeNull()

    cleanup()
    renderShell()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /apply migrations/i }).hasAttribute('disabled')).toBe(false)
    })
    fireEvent.click(screen.getByRole('button', { name: /apply migrations/i }))
    expect(await screen.findByText('Schema diff content ?apply=1')).not.toBeNull()
  })
})
