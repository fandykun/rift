import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchLint, fetchMigrations, fetchStatus, triggerDown } from '../lib/api'
import { useAppStore } from '../stores/appStore'
import { MigrationsPage } from './MigrationsPage'

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchStatus: vi.fn(),
    fetchMigrations: vi.fn(),
    fetchLint: vi.fn(),
    triggerDown: vi.fn(),
  }
})

function renderWithProviders() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MigrationsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('MigrationsPage', () => {
  beforeEach(() => {
    localStorage.clear()
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false })
    vi.mocked(fetchStatus).mockResolvedValue({
      environment: 'test',
      counts: { total: 2, applied: 1, pending: 1, rolled_back: 0 },
    })
    vi.mocked(fetchMigrations).mockResolvedValue([
      {
        version: '20260621_120000',
        filename: '20260621_120000_create_accounts.up.sql',
        status: 'pending',
        applied_by: 'local',
        has_lint: true,
      },
      {
        version: '20260620_180000',
        filename: '20260620_180000_create_widgets.up.sql',
        status: 'applied',
        applied_at: '2026-06-20T10:00:00Z',
        applied_by: 'ci',
        execution_ms: 42,
        has_lint: false,
      },
    ])
    vi.mocked(fetchLint).mockResolvedValue({
      error_count: 1,
      warning_count: 0,
      results: [
        {
          filename: '20260621_120000_create_accounts.up.sql',
          warnings: [
            {
              pattern: 'DROP_COLUMN',
              line: 3,
              severity: 'error',
              message: 'Dropping a column is irreversible.',
              suggestion: 'Rename the column first.',
            },
          ],
        },
      ],
    })
    vi.mocked(triggerDown).mockResolvedValue({ status: 'rolled-back', steps: 1 })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('renders migration rows, counts, linter alerts, and action links from mocked API data', async () => {
    renderWithProviders()

    expect(await screen.findByText('create_accounts')).not.toBeNull()
    expect(screen.getByText('create_widgets')).not.toBeNull()
    expect(screen.getByText('Total Migrations')).not.toBeNull()
    expect(screen.getByText('Applied')).not.toBeNull()
    expect(screen.getByText('Pending')).not.toBeNull()
    expect(screen.getByText('DROP_COLUMN')).not.toBeNull()
    expect(screen.getAllByRole('link', { name: 'View' })[0]?.getAttribute('href')).toBe('/migrations/20260621_120000')
    expect(screen.getAllByRole('link', { name: 'Diff' })[0]?.getAttribute('href')).toBe('/migrations/20260621_120000/diff')

    await waitFor(() => expect(useAppStore.getState().environmentName).toBe('test'))
  })

  it('renders the empty-state guidance when the API returns no migrations', async () => {
    vi.mocked(fetchStatus).mockResolvedValue({
      environment: 'test',
      counts: { total: 0, applied: 0, pending: 0, rolled_back: 0 },
    })
    vi.mocked(fetchMigrations).mockResolvedValue([])
    vi.mocked(fetchLint).mockResolvedValue({ error_count: 0, warning_count: 0, results: [] })

    renderWithProviders()

    expect(await screen.findByText('No migrations yet')).not.toBeNull()
    expect(screen.getByText('rift new add_users')).not.toBeNull()
  })

  it('checks database connectivity from the quick action button', async () => {
    renderWithProviders()

    await screen.findByText('create_accounts')
    fireEvent.click(screen.getByRole('button', { name: /connect to db/i }))

    await waitFor(() => expect(fetchStatus).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('Database connection OK · test')).not.toBeNull()
  })

  it('syncs local migration files by refetching dashboard data', async () => {
    renderWithProviders()

    await screen.findByText('create_accounts')
    fireEvent.click(screen.getByRole('button', { name: /sync local files/i }))

    await waitFor(() => {
      expect(fetchStatus).toHaveBeenCalledTimes(2)
      expect(fetchMigrations).toHaveBeenCalledTimes(2)
      expect(fetchLint).toHaveBeenCalledTimes(2)
    })
    expect(await screen.findByText('Synced 2 migrations · 1 lint finding')).not.toBeNull()
  })

  it('confirms and rolls back only the latest applied migration', async () => {
    renderWithProviders()

    await screen.findByText('create_widgets')
    fireEvent.click(screen.getByRole('button', { name: /rollback create_widgets/i }))

    expect(screen.getByRole('heading', { name: 'Rollback latest migration?' })).not.toBeNull()
    expect(screen.getByText('20260620_180000_create_widgets.up.sql')).not.toBeNull()
    expect(triggerDown).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Confirm rollback' }))

    await waitFor(() => {
      expect(triggerDown).toHaveBeenCalledWith(1, 'token')
      expect(fetchStatus).toHaveBeenCalledTimes(2)
      expect(fetchMigrations).toHaveBeenCalledTimes(2)
      expect(fetchLint).toHaveBeenCalledTimes(2)
    })
    expect(await screen.findByText('Rolled back create_widgets.')).not.toBeNull()
  })

  it('shows rollback errors without refreshing stale dashboard data', async () => {
    vi.mocked(triggerDown).mockRejectedValue(new Error('down migration failed'))
    renderWithProviders()

    await screen.findByText('create_widgets')
    fireEvent.click(screen.getByRole('button', { name: /rollback create_widgets/i }))
    fireEvent.click(screen.getByRole('button', { name: 'Confirm rollback' }))

    expect(await screen.findByText('down migration failed')).not.toBeNull()
    expect(fetchStatus).toHaveBeenCalledTimes(1)
    expect(fetchMigrations).toHaveBeenCalledTimes(1)
    expect(fetchLint).toHaveBeenCalledTimes(1)
  })
})
