export type GraphQLError = { message: string }

export type GraphQLResponse<T> = {
  data?: T
  errors?: GraphQLError[]
}

const tokenStorageKey = 'products-auth-token'

export function getAuthToken(): string | null {
  return localStorage.getItem(tokenStorageKey)
}

export function setAuthToken(token: string): void {
  localStorage.setItem(tokenStorageKey, token)
}

export function clearAuthToken(): void {
  localStorage.removeItem(tokenStorageKey)
}

export async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const token = getAuthToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch('/graphql', {
    method: 'POST',
    headers,
    body: JSON.stringify({ query, variables }),
  })

  const payload = (await response.json()) as GraphQLResponse<T>

  if (!response.ok) {
    if (payload.errors?.length) {
      throw new Error(payload.errors[0].message)
    }
    throw new Error(`GraphQL request failed (${response.status})`)
  }

  if (payload.errors?.length) {
    throw new Error(payload.errors[0].message)
  }
  if (!payload.data) {
    throw new Error('No data returned from GraphQL API')
  }
  return payload.data
}
