type StatCardProps = {
  label: string
  value: number | string
  icon: string
}

export function StatCard({ label, value, icon }: StatCardProps) {
  return (
    <section className="group relative flex flex-col gap-unit-2 overflow-hidden rounded border border-outline-variant bg-surface-container p-unit-4">
      <span className="font-label-caps text-label-caps uppercase text-on-surface-variant">{label}</span>
      <strong className="font-headline-md text-headline-md text-on-background">{value}</strong>
      <span className="material-symbols-outlined absolute right-0 top-0 p-unit-4 text-4xl opacity-20 transition-opacity group-hover:opacity-40">
        {icon}
      </span>
    </section>
  )
}
