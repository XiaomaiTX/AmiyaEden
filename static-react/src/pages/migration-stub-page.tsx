interface MigrationStubPageProps {
  title: string
  path: string
  batch: 'A' | 'B' | 'C' | 'D' | 'Tail'
}

export function MigrationStubPage({ title, path, batch }: MigrationStubPageProps) {
  return (
    <section className="rounded-lg border bg-card p-5">
      <h1 className="text-xl font-semibold">{title}</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Route: <code>{path}</code>
      </p>
      <p className="mt-1 text-sm text-muted-foreground">
        Migration batch: <code>{batch}</code>
      </p>
      <p className="mt-4 text-sm">
        This page is a React migration stub. Vue behavior and API integration will be filled in by
        batch implementation.
      </p>
    </section>
  )
}
