import { useState, useMemo } from 'react'
import { useParams } from 'react-router-dom'

function decodeIframeSrc(splat: string): string {
  const trimmed = splat.replace(/^\/+|\/+$/g, '')
  if (!trimmed) return ''

  let decoded: string
  try {
    decoded = decodeURIComponent(trimmed)
  } catch {
    return ''
  }

  let parsed: URL
  try {
    parsed = new URL(decoded)
  } catch {
    return ''
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return ''
  }

  return parsed.toString()
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
        referrerPolicy="no-referrer"
        onLoad={() => setLoaded(true)}
      />
    </div>
  )
}
