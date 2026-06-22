import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type AppState = {
  apiToken: string
  environmentName: string
  theme: 'light' | 'dark'
  sidebarOpen: boolean
  setApiToken: (apiToken: string) => void
  setEnvironmentName: (environmentName: string) => void
  toggleTheme: () => void
  setSidebarOpen: (sidebarOpen: boolean) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      apiToken: '',
      environmentName: 'development',
      theme: 'dark',
      sidebarOpen: false,
      setApiToken: (apiToken) => set({ apiToken }),
      setEnvironmentName: (environmentName) => set({ environmentName }),
      toggleTheme: () => set((state) => ({ theme: state.theme === 'dark' ? 'light' : 'dark' })),
      setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
    }),
    { name: 'rift-app-state' },
  ),
)
