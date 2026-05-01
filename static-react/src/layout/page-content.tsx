import { Outlet } from 'react-router-dom'

export function PageContent() {
  return (
    <main className="min-h-0 flex-1 overflow-auto p-4 md:p-6">
      <div className="mx-auto w-full max-w-6xl">
        <Outlet />
      </div>
    </main>
  )
}
