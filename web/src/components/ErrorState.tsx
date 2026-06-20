type ErrorStateProps = {
  message: string
  onRetry?: () => void
}

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <section className="rounded border border-error/50 bg-error-container/10 p-unit-4 text-error">
      <div className="flex items-start gap-unit-2">
        <span className="material-symbols-outlined">error</span>
        <div>
          <h3 className="font-headline-sm text-headline-sm">Something went wrong</h3>
          <p className="mt-unit-2 font-code-sm text-code-sm text-on-error-container">{message}</p>
          {onRetry ? (
            <button
              className="mt-unit-4 rounded border border-error/30 bg-error/10 px-3 py-1.5 font-label-caps text-label-caps uppercase text-error hover:bg-error/20"
              type="button"
              onClick={onRetry}
            >
              Retry
            </button>
          ) : null}
        </div>
      </div>
    </section>
  )
}
