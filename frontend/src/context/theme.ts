import { createContext } from 'react'

export type Theme = 'light' | 'dark'

export interface ThemeState {
  theme: Theme
  toggleTheme: () => void
}

export const ThemeContext = createContext<ThemeState | null>(null)
