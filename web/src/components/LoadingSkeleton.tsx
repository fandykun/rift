export function LoadingSkeleton() {
  return (
    <div className="space-y-unit-4" aria-label="Loading">
      <div className="grid gap-unit-4 md:grid-cols-3">
        {[0, 1, 2].map((item) => (
          <div key={item} className="h-24 animate-pulse rounded bg-surface-container-high" />
        ))}
      </div>
      <div className="rounded border border-outline-variant bg-surface-container p-unit-4">
        {[0, 1, 2, 3].map((item) => (
          <div key={item} className="mb-unit-4 h-8 animate-pulse rounded bg-surface-container-high last:mb-0" />
        ))}
      </div>
    </div>
  )
}
