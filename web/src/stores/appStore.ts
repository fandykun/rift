import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type AppState = {
  apiToken: string
  environmentName: string
  sidebarOpen: boolean
  setApiToken: (apiToken: string) => void
  setEnvironmentName: (environmentName: string) => void
  setSidebarOpen: (sidebarOpen: boolean) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      apiToken: '',
      environmentName: 'development',
      sidebarOpen: false,
      setApiToken: (apiToken) => set({ apiToken }),
      setEnvironmentName: (environmentName) => set({ environmentName }),
      setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
    }),
    { name: 'rift-app-state' },
  ),
)
