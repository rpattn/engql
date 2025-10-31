import { createContext, useCallback, useContext, useMemo } from 'react'

import { useOrganizations } from '@/features/organizations'

type TransformationsContextValue = {
  organizationId: string
  setOrganizationId: (organizationId: string) => void
}

const TransformationsContext = createContext<TransformationsContextValue | undefined>(
  undefined,
)

export function TransformationsProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const { selectedOrganizationId, setSelectedOrganizationId } = useOrganizations()

  const setOrganizationId = useCallback(
    (organizationId: string) => {
      const normalized = organizationId.trim()
      setSelectedOrganizationId(normalized.length > 0 ? normalized : null)
    },
    [setSelectedOrganizationId],
  )

  const value = useMemo<TransformationsContextValue>(
    () => ({ organizationId: selectedOrganizationId ?? '', setOrganizationId }),
    [selectedOrganizationId, setOrganizationId],
  )

  return (
    <TransformationsContext.Provider value={value}>
      {children}
    </TransformationsContext.Provider>
  )
}

export function useTransformationsContext() {
  const ctx = useContext(TransformationsContext)

  if (!ctx) {
    throw new Error('useTransformationsContext must be used within a TransformationsProvider')
  }

  return ctx
}
