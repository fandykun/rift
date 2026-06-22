import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SQLDiffPane } from './SQLDiffPane'

describe('SQLDiffPane', () => {
  it('renders diff pane headers, line numbers, and line treatments', () => {
    render(
      <SQLDiffPane
        dot="local"
        subtitle="20260621_120000_create_accounts"
        title="LOCAL MIGRATIONS"
        lines={[
          { number: 1, content: 'CREATE TABLE accounts (', type: 'addition' },
          { number: 2, content: 'DROP COLUMN legacy_name;', type: 'deletion' },
          { number: 3, content: '/* no live table */', type: 'comment' },
        ]}
      />,
    )

    expect(screen.getByText('LOCAL MIGRATIONS')).not.toBeNull()
    expect(screen.getByText('20260621_120000_create_accounts')).not.toBeNull()
    expect(screen.getByText('1')).not.toBeNull()
    expect(screen.getByText('CREATE')).not.toBeNull()

    const deletedLine = screen.getByText('DROP').closest('div')
    expect(deletedLine).not.toBeNull()
    expect(deletedLine?.className).toContain('line-through')
    expect(screen.getByText('/* no live table */')).not.toBeNull()
  })

  it('renders the empty structural-change state', () => {
    render(<SQLDiffPane dot="live" lines={[]} subtitle="schema: public" title="LIVE DATABASE" />)

    expect(screen.getByText('LIVE DATABASE')).not.toBeNull()
    expect(screen.getByText('/* no structural changes */')).not.toBeNull()
  })
})
