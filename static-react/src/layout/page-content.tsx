import { Outlet } from 'react-router-dom'

export function PageContent() {
  return (
    <main className="h-[calc(100vh-56px)] overflow-auto p-4 md:p-6">
      <div className="mx-auto w-full max-w-6xl">
        <Outlet />
      </div>
    </main>
  )
}
