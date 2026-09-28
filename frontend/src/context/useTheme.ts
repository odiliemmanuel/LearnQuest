import { useContext } from 'react'
import { ThemeContext, type ThemeState } from './theme'

export function useTheme(): ThemeState {
  const context = useContext(ThemeContext)
  if (!context) throw new Error('useTheme must be used within ThemeProvider')
  return context
}
