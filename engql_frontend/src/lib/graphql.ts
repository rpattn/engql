const GRAPHQL_ENDPOINT =
  import.meta.env.VITE_GRAPHQL_URL ?? 'http://localhost:8080/query'

type GraphQLResponse<T> = {
  data?: T
  errors?: Array<{ message: string }>
}

export async function graphqlRequest<TData>(
  query: string,
  variables?: Record<string, unknown>,
): Promise<TData> {
  const response = await fetch(GRAPHQL_ENDPOINT, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      query,
      variables,
    }),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Network error (${response.status}): ${text}`)
  }

  const payload = (await response.json()) as GraphQLResponse<TData>

  if (payload.errors?.length) {
    throw new Error(payload.errors.map((err) => err.message).join('\n'))
  }

  if (!payload.data) {
    throw new Error('GraphQL response was empty')
  }

  return payload.data
}
