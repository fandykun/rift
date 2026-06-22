import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { useAppStore } from '../stores/appStore'
import { AppShell } from './AppShell'

function renderShell() {
  return render(
    <MemoryRouter initialEntries={['/migrations']}>
      <Routes>
        <Route element={<AppShell />} path="/">
          <Route element={<div>Dashboard content</div>} path="migrations" />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

describe('AppShell', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false, theme: 'dark' })
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
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
})
