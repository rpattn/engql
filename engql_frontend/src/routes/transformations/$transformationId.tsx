import { Outlet, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/transformations/$transformationId')({
  component: TransformationDetailLayout,
})

function TransformationDetailLayout() {
  return <Outlet />
}
