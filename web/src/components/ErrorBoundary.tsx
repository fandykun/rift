import { ErrorBoundary as ReactErrorBoundary } from 'react-error-boundary'
import type { ReactNode } from 'react'
import { ErrorState } from './ErrorState'

type ErrorBoundaryProps = {
  children: ReactNode
}

export function ErrorBoundary({ children }: ErrorBoundaryProps) {
  return (
    <ReactErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }) => (
        <ErrorState
          message={error instanceof Error ? error.message : 'Unexpected render failure'}
          onRetry={resetErrorBoundary}
        />
      )}
    >
      {children}
    </ReactErrorBoundary>
  )
}
