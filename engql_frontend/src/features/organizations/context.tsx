import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

import { useGetOrganizationsQuery } from '@/generated/graphql'
import { loadLastOrganizationId, persistLastOrganizationId } from '@/lib/browserStorage'

type OrganizationsContextValue = {
  organizations: { id: string; name: string }[]
  selectedOrganizationId: string | null
  setSelectedOrganizationId: (organizationId: string | null) => void
  isLoading: boolean
  isFetching: boolean
  error: unknown
  refetch: () => Promise<unknown>
}

const OrganizationsContext = createContext<OrganizationsContextValue | undefined>(
  undefined,
)

function getInitialOrganizationId() {
  return loadLastOrganizationId()
}

export function OrganizationsProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const organizationsQuery = useGetOrganizationsQuery()
  const organizations = organizationsQuery.data?.organizations ?? []
  const [selectedOrganizationId, setSelectedOrganizationIdState] = useState<string | null>(
    getInitialOrganizationId,
  )

  useEffect(() => {
    if (organizations.length === 0) {
      setSelectedOrganizationIdState(null)
      return
    }

    setSelectedOrganizationIdState((current) => {
      if (current && organizations.some((org) => org.id === current)) {
        return current
      }

      const stored = loadLastOrganizationId()
      if (stored && organizations.some((org) => org.id === stored)) {
        return stored
      }

      return organizations[0]?.id ?? null
    })
  }, [organizations])

  useEffect(() => {
    persistLastOrganizationId(selectedOrganizationId)
  }, [selectedOrganizationId])

  const setSelectedOrganizationId = useCallback((organizationId: string | null) => {
    setSelectedOrganizationIdState(organizationId)
  }, [])

  const value = useMemo<OrganizationsContextValue>(
    () => ({
      organizations,
      selectedOrganizationId,
      setSelectedOrganizationId,
      isLoading: organizationsQuery.isLoading,
      isFetching: organizationsQuery.isFetching,
      error: organizationsQuery.error ?? null,
      refetch: organizationsQuery.refetch,
    }),
    [
      organizations,
      organizationsQuery.error,
      organizationsQuery.isFetching,
      organizationsQuery.isLoading,
      organizationsQuery.refetch,
      selectedOrganizationId,
      setSelectedOrganizationId,
    ],
  )

  return (
    <OrganizationsContext.Provider value={value}>
      {children}
    </OrganizationsContext.Provider>
  )
}

export function useOrganizations() {
  const context = useContext(OrganizationsContext)

  if (!context) {
    throw new Error('useOrganizations must be used within an OrganizationsProvider')
  }

  return context
}
