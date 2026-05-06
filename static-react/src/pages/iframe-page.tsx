import { useState, useMemo } from 'react'
import { useParams } from 'react-router-dom'

function decodeIframeSrc(splat: string): string {
  const trimmed = splat.replace(/^\/+|\/+$/g, '')
  if (!trimmed) return ''

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return trimmed
  }

  if (trimmed.startsWith('http%3A%2F%2F') || trimmed.startsWith('https%3A%2F%2F')) {
    return decodeURIComponent(trimmed)
  }

  return decodeURIComponent(trimmed)
}

export function IframePage() {
  const { '*': splat } = useParams<{ '*': string }>()
  const [loaded, setLoaded] = useState(false)

  const src = useMemo(() => (splat ? decodeIframeSrc(splat) : ''), [splat])

  if (!src) {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <p className="text-sm text-muted-foreground">
          Missing iframe target path.
        </p>
      </div>
    )
  }

  return (
    <div className="relative h-full w-full">
      {!loaded ? (
        <div className="absolute inset-0 flex items-center justify-center bg-background">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
        </div>
      ) : null}
      <iframe
        src={src}
        title="External Content"
        className="h-full w-full min-h-[calc(100vh-120px)] border-none"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
        referrerPolicy="noopener noreferrer"
        onLoad={() => setLoaded(true)}
      />
    </div>
  )
}
