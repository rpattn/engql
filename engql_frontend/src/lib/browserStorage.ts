const LAST_ORGANIZATION_STORAGE_KEY = 'engql:lastOrganizationId'

export function loadLastOrganizationId(): string | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    return localStorage.getItem(LAST_ORGANIZATION_STORAGE_KEY)
  } catch {
    return null
  }
}

export function persistLastOrganizationId(organizationId: string | null): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    if (organizationId) {
      localStorage.setItem(LAST_ORGANIZATION_STORAGE_KEY, organizationId)
    } else {
      localStorage.removeItem(LAST_ORGANIZATION_STORAGE_KEY)
    }
  } catch {
    // ignore write failures (e.g. private browsing)
  }
}

export function clearLastOrganizationId(): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    localStorage.removeItem(LAST_ORGANIZATION_STORAGE_KEY)
  } catch {
    // ignore failures
  }
}

export { LAST_ORGANIZATION_STORAGE_KEY }
