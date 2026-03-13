import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
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

function mockGraphQL() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init?: RequestInit) => {
      const body = JSON.parse(String(init?.body ?? '{}')) as { query?: string }
      const query = body.query ?? ''

      if (query.includes('query ListProducts')) {
        return { json: async () => listProductsPayload }
      }
      if (query.includes('query ProductById')) {
        return { json: async () => detailPayload }
      }
      if (query.includes('mutation CreateProduct')) {
        return { json: async () => ({ data: { createProduct: { id: '3' } } }) }
      }
      if (query.includes('mutation UpdateProduct')) {
        return { json: async () => ({ data: { updateProduct: { id: '1' } } }) }
      }
      if (query.includes('mutation DeleteProduct')) {
        return { json: async () => ({ data: { deleteProduct: true } }) }
      }

      return { json: async () => ({ data: {} }) }
    }),
  )
}

describe('requirements coverage', () => {
  beforeEach(() => {
    window.location.hash = '#/table'
    mockGraphQL()
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('shows dedicated table-view screen', async () => {
    render(<App />)

    await waitFor(() => expect(screen.getByText('Products table')).toBeTruthy())
    expect(screen.getByText('Table view')).toBeTruthy()
    expect(screen.queryByText('Products by status (kanban)')).toBeNull()

    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeTruthy()
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeTruthy()
    expect(screen.getAllByRole('button', { name: 'MacBook Pro 14' }).length).toBeGreaterThan(0)
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

  it('shows product view with details and shared create/edit page layout', async () => {
    window.location.hash = '#/products/1'
    render(<App />)

    await waitFor(() => expect(screen.getByText('Product details')).toBeTruthy())
    await waitFor(() => expect(screen.getByText('https://example.com/purchase')).toBeTruthy())

    expect(screen.getAllByText('MacBook Pro 14').length).toBeGreaterThan(0)
    expect(screen.getByText('https://example.com/shop')).toBeTruthy()
    expect(screen.getByText('https://example.com/booqable')).toBeTruthy()
    expect(screen.getByText('https://example.com/manual')).toBeTruthy()
    expect(screen.getByText('https://example.com/inspection')).toBeTruthy()

    expect(screen.getAllByText('Create / edit page').length).toBeGreaterThan(0)
  })
})
