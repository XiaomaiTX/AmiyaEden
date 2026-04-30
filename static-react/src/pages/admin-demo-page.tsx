import { useSessionStore } from '@/stores'

export function AdminDemoPage() {
  const roles = useSessionStore((state) => state.roles)
  const authList = useSessionStore((state) => state.authList)

  return (
    <div className="space-y-4">
      <section className="rounded-lg border bg-card p-4">
        <h1 className="text-xl font-semibold">Admin Route Demo</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          此页要求角色 `admin/super_admin`，并消费路由 `authList` 元数据。
        </p>
      </section>

      <section className="rounded-lg border bg-card p-4 text-sm">
        <p className="text-muted-foreground">roles: {roles.length > 0 ? roles.join(', ') : 'none'}</p>
        <p className="text-muted-foreground">
          route authList: {authList.length > 0 ? authList.join(', ') : 'none'}
        </p>
      </section>
    </div>
  )
}