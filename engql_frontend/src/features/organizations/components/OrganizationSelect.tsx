import type { SelectHTMLAttributes } from 'react'
import { useCallback, useMemo } from 'react'

import { useOrganizations } from '../context'

type OrganizationSelectProps = Omit<
  SelectHTMLAttributes<HTMLSelectElement>,
  'value' | 'onChange'
> & {
  value: string | null
  onChange: (value: string | null) => void
  placeholder?: string
}

export function OrganizationSelect({
  value,
  onChange,
  placeholder = 'Select an organization',
  className = '',
  disabled,
  ...rest
}: OrganizationSelectProps) {
  const { organizations, isLoading } = useOrganizations()

  const handleChange = useCallback(
    (event: React.ChangeEvent<HTMLSelectElement>) => {
      const nextValue = event.target.value
      onChange(nextValue ? nextValue : null)
    },
    [onChange],
  )

  const combinedClassName = useMemo(() => {
    const baseClasses =
      'rounded-md border border-subtle bg-surface px-3 py-2 text-sm transition focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200 disabled:cursor-not-allowed disabled:bg-subtle'

    return [baseClasses, className].filter(Boolean).join(' ')
  }, [className])

  const optionLabel = useMemo(() => {
    if (isLoading) {
      return 'Loading organizations…'
    }

    if (organizations.length === 0) {
      return 'No organizations available'
    }

    return placeholder
  }, [isLoading, organizations.length, placeholder])

  const isDisabled =
    disabled !== undefined ? disabled : organizations.length === 0 || isLoading

  return (
    <select
      value={value ?? ''}
      onChange={handleChange}
      className={combinedClassName}
      disabled={isDisabled}
      {...rest}
    >
      <option value="" disabled>
        {optionLabel}
      </option>
      {organizations.map((organization) => (
        <option key={organization.id} value={organization.id}>
          {organization.name}
        </option>
      ))}
    </select>
  )
}
