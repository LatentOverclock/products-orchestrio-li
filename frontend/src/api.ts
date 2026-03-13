export type GraphQLError = { message: string }

export type GraphQLResponse<T> = {
  data?: T
  errors?: GraphQLError[]
}

export async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const response = await fetch('/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })

  const payload = (await response.json()) as GraphQLResponse<T>
  if (payload.errors?.length) {
    throw new Error(payload.errors[0].message)
  }
  if (!payload.data) {
    throw new Error('No data returned from GraphQL API')
  }
  return payload.data
}
