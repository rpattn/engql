import { Link, Outlet, createFileRoute, useLocation } from '@tanstack/react-router'
import { useMemo } from 'react'

import { TransformationsProvider, useTransformationsContext } from '@/features/transformations/context'

export const Route = createFileRoute('/transformations')({
  component: TransformationsLayout,
})

function TransformationsLayout() {
  return (
    <TransformationsProvider>
      <LayoutShell />
    </TransformationsProvider>
  )
}

function LayoutShell() {
  const { organizationId, setOrganizationId } = useTransformationsContext()
  const location = useLocation()

  const tabs = useMemo(
    () => [
      {
        label: 'Catalog',
        to: '/transformations',
        active: location.pathname === '/transformations',
      },
    ],
    [location.pathname],
  )

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6 px-6 py-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-slate-900">Entity transformations</h1>
          <p className="text-sm text-slate-500">
            Design, manage, and execute transformation DAGs for your organization.
          </p>
        </div>
        <div>
          <label className="block text-xs font-semibold text-slate-600">
            Organization ID
            <input
              value={organizationId}
              placeholder="org_123"
              onChange={(event) => setOrganizationId(event.target.value)}
              className="mt-1 w-56 rounded border border-slate-300 px-3 py-1 text-sm"
            />
          </label>
        </div>
      </header>

      <nav className="flex gap-2">
        {tabs.map((tab) => (
          <Link
            key={tab.to}
            to={tab.to}
            className={`rounded px-3 py-1 text-xs font-semibold ${
              tab.active
                ? 'bg-blue-600 text-white'
                : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
            }`}
          >
            {tab.label}
          </Link>
        ))}
      </nav>

      <Outlet />
    </div>
  )
}
