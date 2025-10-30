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
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-8">
      <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
        <header className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold">Entity transformations</h1>
            <p className="mt-1 text-sm text-muted">
              Design, manage, and execute transformation DAGs for your organization.
            </p>
          </div>
          <label className="block text-xs font-semibold text-muted">
            Organization
            <OrganizationSelect
              value={organizationId || null}
              onChange={(value) => setOrganizationId(value ?? '')}
              className="mt-1 w-56"
            />
          </label>
        </header>

        <nav className="mt-6 flex gap-2">
          {tabs.map((tab) => (
            <Link
              key={tab.to}
              to={tab.to}
              className={`rounded-md px-4 py-2 text-xs font-semibold transition ${
                tab.active
                  ? 'bg-blue-600 text-white shadow-sm'
                  : 'border border-subtle text-muted hover:border-blue-500/60 hover:text-blue-500'
              }`}
            >
              {tab.label}
            </Link>
          ))}
        </nav>
      </section>

      <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
        <Outlet />
      </section>
    </div>
  )
}
