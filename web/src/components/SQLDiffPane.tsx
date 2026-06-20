import type { ReactNode } from 'react'

export type SQLDiffLine = {
  number: number
  content: string
  type: 'unchanged' | 'addition' | 'deletion' | 'comment'
}

type SQLDiffPaneProps = {
  title: string
  subtitle: string
  dot: 'local' | 'live'
  lines: SQLDiffLine[]
}

export function SQLDiffPane({ title, subtitle, dot, lines }: SQLDiffPaneProps) {
  return (
    <section className="flex min-h-0 flex-1 flex-col bg-surface-container-lowest">
      <div className="flex items-center justify-between border-b border-outline-variant bg-surface-container-high px-unit-4 py-2">
        <div className="flex items-center gap-unit-2 font-label-caps text-label-caps uppercase text-on-surface-variant">
          <span className={dot === 'local' ? 'h-2 w-2 animate-pulse rounded-full bg-primary' : 'h-2 w-2 rounded-full border border-outline'} />
          {title}
        </div>
        <span className={dot === 'local' ? 'font-code-sm text-code-sm text-primary' : 'font-code-sm text-code-sm text-on-surface-variant'}>
          {subtitle}
        </span>
      </div>
      <div className="min-h-0 flex-1 overflow-auto py-unit-4 font-code-sm text-code-sm">
        {lines.length > 0 ? (
          lines.map((line) => <SQLLine key={`${line.number}-${line.type}-${line.content}`} line={line} />)
        ) : (
          <div className="px-unit-4 text-on-surface-variant italic">/* no structural changes */</div>
        )}
      </div>
    </section>
  )
}

function SQLLine({ line }: { line: SQLDiffLine }) {
  const rowClass = {
    unchanged: 'text-on-surface',
    addition: 'bg-secondary/5 text-secondary',
    deletion: 'bg-error/5 text-error line-through opacity-70',
    comment: 'text-on-surface-variant italic',
  }[line.type]

  return (
    <div className={`grid grid-cols-[4rem_minmax(0,1fr)] px-unit-4 ${rowClass}`}>
      <span className="select-none pr-unit-4 text-right font-code-sm text-code-sm text-on-surface-variant/40">
        {line.number}
      </span>
      <code className="whitespace-pre-wrap break-words">{highlightSQL(line.content)}</code>
    </div>
  )
}

function highlightSQL(content: string): ReactNode[] {
  if (content.trim().startsWith('/*') || content.trim().startsWith('--')) {
    return [
      <span key="comment" className="text-outline italic">
        {content}
      </span>,
    ]
  }

  const pattern = /(CREATE|ALTER|TABLE|ADD|DROP|COLUMN|INDEX|UNIQUE|PRIMARY|KEY|REFERENCES|NOT|NULL|DEFAULT|BIGSERIAL|BIGINT|INTEGER|TEXT|UUID|VARCHAR|TIMESTAMP|BOOLEAN|ON|CONSTRAINT|FOREIGN|IF|EXISTS|CONCURRENTLY|'[^']*'|\b\d+\b|[(),;])/gi
  const parts = content.split(pattern).filter((part) => part.length > 0)
  return parts.map((part, index) => {
    const upper = part.toUpperCase()
    if (/^'.*'$/.test(part)) {
      return <span key={`${part}-${index}`} className="text-secondary-container">{part}</span>
    }
    if (/^\d+$/.test(part)) {
      return <span key={`${part}-${index}`} className="text-tertiary">{part}</span>
    }
    if (/^[(),;]$/.test(part)) {
      return <span key={`${part}-${index}`} className="text-on-surface-variant">{part}</span>
    }
    if (['BIGSERIAL', 'BIGINT', 'INTEGER', 'TEXT', 'UUID', 'VARCHAR', 'TIMESTAMP', 'BOOLEAN'].includes(upper)) {
      return <span key={`${part}-${index}`} className="text-tertiary">{part}</span>
    }
    if (['CREATE', 'ALTER', 'TABLE', 'ADD', 'DROP', 'COLUMN', 'INDEX', 'UNIQUE', 'PRIMARY', 'KEY', 'REFERENCES', 'NOT', 'NULL', 'DEFAULT', 'ON', 'CONSTRAINT', 'FOREIGN', 'IF', 'EXISTS', 'CONCURRENTLY'].includes(upper)) {
      return <span key={`${part}-${index}`} className="font-semibold text-primary">{part}</span>
    }
    return <span key={`${part}-${index}`}>{part}</span>
  })
}
