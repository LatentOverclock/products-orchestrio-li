import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { App } from '../src/App'

const listProductsPayload = {
  data: {
    products: [
      {
        id: '1',
        name: 'MacBook Pro 14',
        purchaseLink: 'https://example.com/purchase',
        shopLink: 'https://example.com/shop',
        booqableLink: 'https://example.com/booqable',
        manualLink: 'https://example.com/manual',
        inspectionLink: 'https://example.com/inspection',
        description: 'Main laptop',
        status: 'mafo',
        createdAt: '2026-03-13T13:00:00Z',
        updatedAt: '2026-03-13T13:00:00Z',
      },
      {
        id: '2',
        name: 'ThinkPad X1',
        purchaseLink: '',
        shopLink: '',
        booqableLink: '',
        manualLink: '',
        inspectionLink: '',
        description: 'Manual in progress',
        status: 'write-manual',
        createdAt: '2026-03-13T13:10:00Z',
        updatedAt: '2026-03-13T13:10:00Z',
      },
    ],
  },
}

const detailPayload = {
  data: {
    product: {
      id: '1',
      name: 'MacBook Pro 14',
      purchaseLink: 'https://example.com/purchase',
      shopLink: 'https://example.com/shop',
      booqableLink: 'https://example.com/booqable',
      manualLink: 'https://example.com/manual',
      inspectionLink: 'https://example.com/inspection',
      description: 'Main laptop',
      status: 'mafo',
      createdAt: '2026-03-13T13:00:00Z',
      updatedAt: '2026-03-13T13:05:00Z',
    },
  },
}

const usersPayload = {
  data: {
    users: [
      {
        id: '10',
        email: 'admin@example.com',
        createdAt: '2026-03-13T12:00:00Z',
        updatedAt: '2026-03-13T12:00:00Z',
      },
      {
        id: '11',
        email: 'editor@example.com',
        createdAt: '2026-03-13T12:30:00Z',
        updatedAt: '2026-03-13T12:30:00Z',
      },
    ],
  },
}

let fetchMock: ReturnType<typeof vi.fn>

function installFetchMock() {
  fetchMock = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    const urlValue = typeof url === 'string' ? url : url instanceof URL ? url.toString() : url.url

    if (urlValue.endsWith('/login')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ token: 'test-token', user: { email: 'admin@example.com' } }),
      }
    }

    const body = JSON.parse(String(init?.body ?? '{}')) as { query?: string }
    const query = body.query ?? ''

    if (query.includes('query ListProducts')) {
      return { ok: true, status: 200, json: async () => listProductsPayload }
    }
    if (query.includes('query ProductById')) {
      return { ok: true, status: 200, json: async () => detailPayload }
    }
    if (query.includes('query ListUsers')) {
      return { ok: true, status: 200, json: async () => usersPayload }
    }
    if (query.includes('mutation CreateProduct')) {
      return { ok: true, status: 200, json: async () => ({ data: { createProduct: { id: '3' } } }) }
    }
    if (query.includes('mutation UpdateProduct')) {
      return { ok: true, status: 200, json: async () => ({ data: { updateProduct: { id: '1' } } }) }
    }
    if (query.includes('mutation DeleteProduct')) {
      return { ok: true, status: 200, json: async () => ({ data: { deleteProduct: true } }) }
    }
    if (query.includes('mutation UpdateUser')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ data: { updateUser: { id: '10', email: 'updated@example.com' } } }),
      }
    }
    if (query.includes('mutation DeleteUser')) {
      return { ok: true, status: 200, json: async () => ({ data: { deleteUser: true } }) }
    }

    return { ok: true, status: 200, json: async () => ({ data: {} }) }
  })

  vi.stubGlobal('fetch', fetchMock)
}

describe('requirements coverage', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('products-auth-token', 'test-token')
    window.location.hash = '#/table'
    installFetchMock()
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('requires login before showing protected app when no token is present', async () => {
    localStorage.removeItem('products-auth-token')
    render(<App />)

    expect(screen.getByText('Login required')).toBeTruthy()
    expect(screen.getByLabelText('Email')).toBeTruthy()
    expect(screen.getByLabelText('Password')).toBeTruthy()
    expect(screen.queryByText('Products table')).toBeNull()
  })

  it('shows dedicated table-view screen with all product properties except description', async () => {
    render(<App />)

    await waitFor(() => expect(screen.getByText('Products table')).toBeTruthy())

    expect(screen.getByRole('columnheader', { name: /Name/i })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: /Purchase link/i })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: /Shop link/i })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: /Booqable link/i })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: /Manual link/i })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: /Inspection link/i })).toBeTruthy()
    expect(screen.getAllByRole('columnheader', { name: /Status/i }).length).toBeGreaterThan(0)
    expect(screen.queryByRole('columnheader', { name: /Description/i })).toBeNull()

    expect(screen.getByLabelText('Filter name')).toBeTruthy()
    expect(screen.getByLabelText('Filter purchase link')).toBeTruthy()
    expect(screen.getByLabelText('Filter shop link')).toBeTruthy()
    expect(screen.getByLabelText('Filter booqable link')).toBeTruthy()
    expect(screen.getByLabelText('Filter manual link')).toBeTruthy()
    expect(screen.getByLabelText('Filter inspection link')).toBeTruthy()
    expect(screen.getByLabelText('Filter status')).toBeTruthy()
  })

  it('supports sorting and filtering by table properties', async () => {
    const { container } = render(<App />)

    await waitFor(() => expect(screen.getByText('Products table')).toBeTruthy())

    fireEvent.change(screen.getByLabelText('Filter status'), { target: { value: 'mafo' } })
    expect(screen.queryByRole('button', { name: 'ThinkPad X1' })).toBeNull()

    fireEvent.change(screen.getByLabelText('Filter status'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Filter purchase link'), { target: { value: 'purchase' } })
    expect(screen.queryByRole('button', { name: 'ThinkPad X1' })).toBeNull()

    fireEvent.change(screen.getByLabelText('Filter purchase link'), { target: { value: '' } })

    fireEvent.click(screen.getByRole('button', { name: /Name/i }))
    const firstRowNameButton = container.querySelector('tbody tr td .link-button') as HTMLButtonElement | null
    expect(firstRowNameButton?.textContent).toBe('ThinkPad X1')
  })

  it('shows dedicated kanban-view screen', async () => {
    window.location.hash = '#/kanban'
    render(<App />)

    await waitFor(() => expect(screen.getByText('Products by status (kanban)')).toBeTruthy())
    expect(screen.queryByText('Products table')).toBeNull()

    expect(screen.getAllByText('mafo').length).toBeGreaterThan(0)
    expect(screen.getAllByText('write-manual').length).toBeGreaterThan(0)
    expect(screen.getAllByText('all-done').length).toBeGreaterThan(0)
  })

  it('shows unified view/edit page without duplicated detail fields', async () => {
    window.location.hash = '#/products/1'
    render(<App />)

    await waitFor(() => expect(screen.getByText('View / edit page')).toBeTruthy())

    expect((screen.getByLabelText('Name') as HTMLInputElement).value).toBe('MacBook Pro 14')
    expect((screen.getByLabelText('Purchase link') as HTMLInputElement).value).toBe('https://example.com/purchase')
    expect((screen.getByLabelText('Shop link') as HTMLInputElement).value).toBe('https://example.com/shop')
    expect((screen.getByLabelText('Booqable link') as HTMLInputElement).value).toBe('https://example.com/booqable')
    expect((screen.getByLabelText('Manual link') as HTMLInputElement).value).toBe('https://example.com/manual')
    expect((screen.getByLabelText('Inspection link') as HTMLInputElement).value).toBe('https://example.com/inspection')
    expect((screen.getByLabelText('Description') as HTMLTextAreaElement).value).toBe('Main laptop')
    expect((screen.getByLabelText('Status') as HTMLSelectElement).value).toBe('mafo')

    expect(screen.queryByText('Product details')).toBeNull()
    expect(screen.getByText('Product metadata')).toBeTruthy()
  })

  it('allows listing, editing, and deleting users', async () => {
    window.location.hash = '#/users'
    render(<App />)

    await waitFor(() => expect(screen.getByText('Users management')).toBeTruthy())

    expect(screen.getByLabelText('User email 10')).toBeTruthy()
    expect(screen.getByLabelText('User email 11')).toBeTruthy()

    fireEvent.change(screen.getByLabelText('User email 10'), { target: { value: 'updated@example.com' } })
    fireEvent.change(screen.getByLabelText('User password 10'), { target: { value: 'newsecret123' } })

    fireEvent.click(screen.getAllByRole('button', { name: 'Save' })[0])
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[0])

    await waitFor(() => {
      const queries = fetchMock.mock.calls
        .map((call) => String((JSON.parse(String(call[1]?.body ?? '{}')) as { query?: string }).query ?? ''))
        .join('\n')
      expect(queries.includes('mutation UpdateUser')).toBe(true)
      expect(queries.includes('mutation DeleteUser')).toBe(true)
    })
  })
})
