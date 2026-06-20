import { Link } from 'react-router-dom'
import type { LintResult } from '../lib/api'

type LinterAlertsCardProps = {
  results: LintResult[]
}

export function LinterAlertsCard({ results }: LinterAlertsCardProps) {
  const warnings = results.flatMap((result) =>
    result.warnings.map((warning) => ({ ...warning, filename: result.filename })),
  )
  const mostRecent = warnings[0]

  return (
    <section className="rounded border border-outline-variant bg-surface-container/80 p-unit-4 shadow-[inset_0_0_20px_rgba(255,185,95,0.05)] backdrop-blur-sm">
      <div className="flex items-center justify-between">
        <h3 className="font-headline-sm text-headline-sm text-on-surface">Linter Alerts</h3>
        <span className={mostRecent ? 'font-code-sm text-code-sm text-error' : 'font-code-sm text-code-sm text-secondary'}>
          {warnings.length} Issues
        </span>
      </div>

      {mostRecent ? (
        <div className="relative mt-unit-4 overflow-hidden rounded border border-error/50 bg-surface-container-high p-3">
          <div className="absolute left-0 top-0 h-full w-1 bg-error" />
          <p className="font-body-md text-body-md font-semibold text-error">{mostRecent.pattern}</p>
          <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">{mostRecent.message}</p>
          <p className="mt-unit-2 font-code-sm text-code-sm text-on-surface-variant">{mostRecent.filename}</p>
          <Link
            className="mt-unit-4 inline-flex rounded border border-error/30 bg-error/10 px-3 py-1.5 font-label-caps text-label-caps uppercase text-error hover:bg-error/20"
            to="/migrations"
          >
            Review
          </Link>
        </div>
      ) : (
        <p className="mt-unit-4 font-body-md text-body-md text-on-surface-variant">No dangerous pending migrations.</p>
      )}
    </section>
  )
}
