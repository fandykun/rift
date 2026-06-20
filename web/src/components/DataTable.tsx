import type { ReactNode } from 'react'

type DataTableProps = {
  columns: string[]
  children: ReactNode
}

export function DataTable({ columns, children }: DataTableProps) {
  return (
    <div className="overflow-hidden rounded border border-outline-variant bg-surface-container">
      <table className="w-full border-collapse text-left font-body-md text-body-md">
        <thead className="bg-surface-container-high font-label-caps text-label-caps uppercase text-on-surface-variant">
          <tr>
            {columns.map((column) => (
              <th key={column} className="border-b border-outline-variant px-unit-4 py-3 font-semibold">
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-outline-variant">{children}</tbody>
      </table>
    </div>
  )
}
