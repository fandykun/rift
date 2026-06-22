import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  it('renders an applied status with success styling', () => {
    render(<StatusBadge status="applied" />)

    const badge = screen.getByText('applied').closest('span')
    expect(badge).not.toBeNull()
    expect(badge?.className).toContain('text-secondary')
    expect(badge?.className).toContain('border-secondary/30')
  })

  it('uses danger styling for pending migrations with linter findings', () => {
    render(<StatusBadge status="pending" tone="danger" />)

    const badge = screen.getByText('pending').closest('span')
    expect(badge).not.toBeNull()
    expect(badge?.className).toContain('text-error')
    expect(screen.getByText('warning')).not.toBeNull()
  })
})
