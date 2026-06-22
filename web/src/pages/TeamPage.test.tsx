import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchConflicts, fetchHistory, fetchTeam } from '../lib/api'
import { useAppStore } from '../stores/appStore'
import { TeamPage } from './TeamPage'

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchConflicts: vi.fn(),
    fetchHistory: vi.fn(),
    fetchTeam: vi.fn(),
  }
})

function renderWithProviders() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <TeamPage />
    </QueryClientProvider>,
  )
}

describe('TeamPage', () => {
  beforeEach(() => {
    localStorage.clear()
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false })
    vi.mocked(fetchConflicts).mockResolvedValue([
      {
        Type: 'CHECKSUM_MISMATCH',
        Version: '20260621_120000',
        DatabaseFilename: '20260621_120000_create_accounts',
        LocalFilename: '20260621_120000_create_accounts',
        DatabaseChecksum: 'database-checksum',
        LocalChecksum: 'local-checksum',
        Message: 'Migration checksum differs from the database record.',
      },
    ])
    vi.mocked(fetchHistory).mockResolvedValue([
      {
        ID: 1,
        Version: '20260621_120000',
        Filename: '20260621_120000_create_accounts',
        Checksum: 'checksum',
        AppliedAt: '2026-06-21T12:00:00Z',
        AppliedBy: 'fandy',
        ExecutionMs: 111,
        RolledBack: false,
      },
    ])
    vi.mocked(fetchTeam).mockResolvedValue([
      { name: 'Fandy Admin', email: 'fandy@example.com', role: 'Admin' },
      { name: 'Dana Developer', email: 'dana@example.com', role: 'Developer' },
    ])
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders conflicts, deployment timeline, team access, and webhook controls', async () => {
    renderWithProviders()

    expect(await screen.findByText('Conflict Detection')).not.toBeNull()
    expect(screen.getByText('1 conflict detected')).not.toBeNull()
    expect(screen.getByText('Migration checksum differs from the database record.')).not.toBeNull()
    expect(screen.getByText('20260621_120000_create_accounts')).not.toBeNull()
    expect(screen.getByText('Success')).not.toBeNull()
    expect(screen.getByText('Fandy Admin')).not.toBeNull()
    expect(screen.getByText('Dana Developer')).not.toBeNull()
    expect(screen.getByText('Notifications / Webhooks')).not.toBeNull()
    expect(screen.getByText('Slack')).not.toBeNull()
    expect(screen.getByText('Discord')).not.toBeNull()
  })
})
