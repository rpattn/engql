// src/lib/graphql.ts
const GRAPHQL_ENDPOINT =
  import.meta.env.VITE_GRAPHQL_URL ?? "http://localhost:8080/query";

type GraphQLResponse<T> = {
  data?: T;
  errors?: Array<{ message: string }>;
};

// ✅ Add the optional third "headers" argument
export async function graphqlRequest<TData, TVariables extends Record<string, unknown> = Record<string, unknown>>(
  query: string,
  variables?: TVariables,
  headers?: RequestInit["headers"]
): Promise<TData> {
  const response = await fetch(GRAPHQL_ENDPOINT, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(headers ?? {}),
    },
    body: JSON.stringify({ query, variables }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Network error (${response.status}): ${text}`);
  }

  const payload = (await response.json()) as GraphQLResponse<TData>;

  if (payload.errors?.length) {
    throw new Error(payload.errors.map((e) => e.message).join("\n"));
  }

  if (!payload.data) {
    throw new Error("GraphQL response was empty");
  }

  return payload.data;
}