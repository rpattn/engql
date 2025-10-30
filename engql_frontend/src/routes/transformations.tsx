import { Link, Outlet, createFileRoute, useLocation } from '@tanstack/react-router'
import { useMemo } from 'react'

import { OrganizationSelect } from '@/features/organizations'
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
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-slate-900">Entity transformations</h1>
          <p className="text-sm text-slate-500">
            Design, manage, and execute transformation DAGs for your organization.
          </p>
        </div>
        <div>
          <label className="block text-xs font-semibold text-slate-600">
            Organization
            <OrganizationSelect
              value={organizationId || null}
              onChange={(value) => setOrganizationId(value ?? '')}
              className="mt-1 w-56"
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
