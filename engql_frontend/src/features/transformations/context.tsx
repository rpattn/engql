import { createContext, useContext, useEffect, useMemo, useState } from 'react'

type TransformationsContextValue = {
  organizationId: string
  setOrganizationId: (organizationId: string) => void
}

const TransformationsContext = createContext<TransformationsContextValue | undefined>(
  undefined,
)

const STORAGE_KEY = 'engql.transformations.organizationId'

function getInitialOrganizationId() {
  if (typeof window === 'undefined') {
    return ''
  }

  return window.localStorage.getItem(STORAGE_KEY) ?? ''
}

export function TransformationsProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [organizationId, setOrganizationId] = useState<string>(getInitialOrganizationId)

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }

    window.localStorage.setItem(STORAGE_KEY, organizationId)
  }, [organizationId])

  const value = useMemo<TransformationsContextValue>(
    () => ({ organizationId, setOrganizationId }),
    [organizationId],
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
