import { useMemo } from 'react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { ArrowLeft, Info, Link2 } from 'lucide-react'
import { useEntityDetailQuery } from '@/generated/graphql'
import {
  extractEntityDisplayNameFromProperties,
  formatJsonPreview,
  formatJsonPreviewLimited,
  formatRelative,
  formatTimestamp,
} from '@/features/entities/components/helpers'

export const Route = createFileRoute('/entity/$entityId')({
  component: EntityDetailPage,
})

function EntityDetailPage() {
  const { entityId } = Route.useParams()
  const detailQuery = useEntityDetailQuery({ id: entityId }, { staleTime: 30_000 })
  const entity = detailQuery.data?.entity ?? null

  const headingLabel = useMemo(() => {
    if (!entity) {
      return null
    }

    const fallback = entity.referenceValue?.trim() || entity.entityType || entity.id
    return extractEntityDisplayNameFromProperties(entity.properties, fallback)
  }, [entity])

  const propertiesText = useMemo(() => {
    if (!entity) {
      return null
    }
    const formatted = formatJsonPreview(entity.properties)
    return formatted.length > 0 ? formatted : null
  }, [entity])

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-8">
      <Link
        to="/entities"
        className="flex w-fit items-center gap-2 rounded-md border border-subtle bg-surface px-3 py-2 text-sm font-medium text-muted shadow-sm transition hover:border-blue-500/60 hover:text-blue-500"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to entities
      </Link>

      <div className="space-y-2">
        <h1 className="text-3xl font-semibold">
          {headingLabel ?? 'Entity details'}
        </h1>
        <p className="text-sm text-muted">
          Detailed metadata and relationships for this entity.
        </p>
      </div>

      {detailQuery.isLoading && (
        <div className="rounded-xl border border-subtle bg-surface px-4 py-6 text-sm text-muted shadow-sm">
          Loading entity…
        </div>
      )}

      {detailQuery.isError && (
        <div className="flex items-start gap-2 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-400">
          <Info className="mt-0.5 h-4 w-4" />
          <span>Unable to load the entity right now. Please try again shortly.</span>
        </div>
      )}

      {!detailQuery.isLoading && !detailQuery.isError && !entity && (
        <div className="flex items-start gap-2 rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-500">
          <Info className="mt-0.5 h-4 w-4" />
          <span>The requested entity could not be found.</span>
        </div>
      )}

      {entity && (
        <div className="flex flex-col gap-6">
          <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
            <h2 className="text-lg font-semibold">Overview</h2>
            <dl className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Description label="Entity ID" value={entity.id} />
              <Description label="Schema ID" value={entity.schemaId} />
              <Description label="Entity type" value={entity.entityType} />
              <Description label="Reference" value={entity.referenceValue ?? '—'} />
              <Description label="Path" value={entity.path || '—'} />
              <Description label="Version" value={`v${entity.version}`} />
              <Description
                label="Created"
                value={`${formatTimestamp(entity.createdAt)} · ${formatRelative(entity.createdAt)}`}
              />
              <Description
                label="Updated"
                value={`${formatTimestamp(entity.updatedAt)} · ${formatRelative(entity.updatedAt)}`}
              />
            </dl>
          </section>

          <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
            <h2 className="text-lg font-semibold">Properties</h2>
            {propertiesText ? (
              <pre className="mt-4 max-h-96 overflow-auto rounded-lg bg-subtle px-4 py-3 text-xs text-muted">
                {propertiesText}
              </pre>
            ) : (
              <p className="mt-4 text-sm text-muted">No properties are stored for this entity.</p>
            )}
          </section>

          <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
            <div className="flex items-center gap-2">
              <Link2 className="h-4 w-4 text-blue-500" />
              <h2 className="text-lg font-semibold">Linked entities</h2>
            </div>
            {entity.linkedEntities?.length ? (
              <ul className="mt-4 space-y-3">
                {entity.linkedEntities.map((linked) => {
                  const displayName = extractEntityDisplayNameFromProperties(
                    linked.properties,
                    linked.referenceValue || linked.id,
                  )
                  const preview = formatJsonPreviewLimited(linked.properties, 3)
                  return (
                    <li
                      key={linked.id}
                      className="rounded-xl border border-subtle bg-subtle p-4 text-sm text-muted"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div>
                          <p className="font-semibold text-slate-900">{linked.entityType}</p>
                          <p className="text-xs text-muted">ID: {linked.id}</p>
                          {linked.referenceValue && (
                            <p className="text-xs text-muted">Reference: {linked.referenceValue}</p>
                          )}
                          {displayName && (
                            <p className="text-xs text-muted">Display: {displayName}</p>
                          )}
                        </div>
                        <Link
                          to="/entity/$entityId"
                          params={{ entityId: linked.id }}
                          className="rounded-md border border-subtle px-3 py-1 text-xs font-medium text-muted transition hover:border-blue-500/60 hover:text-blue-500"
                        >
                          View details
                        </Link>
                      </div>
                      {preview && (
                        <pre className="mt-3 max-h-40 overflow-auto rounded-lg bg-surface px-3 py-2 text-xs text-muted">
                          {preview}
                        </pre>
                      )}
                    </li>
                  )
                })}
              </ul>
            ) : (
              <p className="mt-4 text-sm text-muted">No linked entities were returned for this entity.</p>
            )}
          </section>
        </div>
      )}
    </div>
  )
}

type DescriptionProps = {
  label: string
  value: string
}

function Description({ label, value }: DescriptionProps) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-wide text-muted">{label}</dt>
      <dd className="mt-1 text-sm text-slate-900">{value}</dd>
    </div>
  )
}
