import { useEffect, useMemo, useState } from 'react'
import { gql } from './api'

type ProductStatus = 'mafo' | 'write-manual' | 'all-done'

type Product = {
  id: string
  name: string
  purchaseLink?: string | null
  shopLink?: string | null
  booqableLink?: string | null
  manualLink?: string | null
  inspectionLink?: string | null
  description: string
  status: ProductStatus
  createdAt: string
  updatedAt: string
}

type ProductInput = {
  name: string
  purchaseLink: string
  shopLink: string
  booqableLink: string
  manualLink: string
  inspectionLink: string
  description: string
  status: ProductStatus
}

const statuses: Array<{ value: ProductStatus; label: string }> = [
  { value: 'mafo', label: 'mafo' },
  { value: 'write-manual', label: 'write-manual' },
  { value: 'all-done', label: 'all-done' },
]

const emptyInput: ProductInput = {
  name: '',
  purchaseLink: '',
  shopLink: '',
  booqableLink: '',
  manualLink: '',
  inspectionLink: '',
  description: '',
  status: 'mafo',
}

const listQuery = `
query ListProducts {
  products {
    id
    name
    purchaseLink
    shopLink
    booqableLink
    manualLink
    inspectionLink
    description
    status
    createdAt
    updatedAt
  }
}`

const detailQuery = `
query ProductById($id: ID!) {
  product(id: $id) {
    id
    name
    purchaseLink
    shopLink
    booqableLink
    manualLink
    inspectionLink
    description
    status
    createdAt
    updatedAt
  }
}`

const createMutation = `
mutation CreateProduct($input: ProductInput!) {
  createProduct(input: $input) { id }
}`

const updateMutation = `
mutation UpdateProduct($id: ID!, $input: ProductInput!) {
  updateProduct(id: $id, input: $input) { id }
}`

const deleteMutation = `
mutation DeleteProduct($id: ID!) {
  deleteProduct(id: $id)
}`

function parseProductIdFromHash(hash: string): string | null {
  const match = hash.match(/^#\/products\/(\d+)$/)
  return match?.[1] ?? null
}

function toGraphQLInput(input: ProductInput) {
  return {
    name: input.name,
    purchaseLink: input.purchaseLink,
    shopLink: input.shopLink,
    booqableLink: input.booqableLink,
    manualLink: input.manualLink,
    inspectionLink: input.inspectionLink,
    description: input.description,
    status: input.status,
  }
}

function fromProduct(product: Product): ProductInput {
  return {
    name: product.name,
    purchaseLink: product.purchaseLink ?? '',
    shopLink: product.shopLink ?? '',
    booqableLink: product.booqableLink ?? '',
    manualLink: product.manualLink ?? '',
    inspectionLink: product.inspectionLink ?? '',
    description: product.description,
    status: product.status,
  }
}

function ProductForm(props: {
  title: string
  submitLabel: string
  value: ProductInput
  onChange: (next: ProductInput) => void
  onSubmit: () => Promise<void>
}) {
  const { title, submitLabel, value, onChange, onSubmit } = props

  const set = <K extends keyof ProductInput>(key: K, next: ProductInput[K]) => {
    onChange({ ...value, [key]: next })
  }

  return (
    <section className="card">
      <h2>{title}</h2>
      <div className="form-grid">
        <label>
          Name
          <input value={value.name} onChange={(e) => set('name', e.target.value)} />
        </label>
        <label>
          Purchase link
          <input value={value.purchaseLink} onChange={(e) => set('purchaseLink', e.target.value)} />
        </label>
        <label>
          Shop link
          <input value={value.shopLink} onChange={(e) => set('shopLink', e.target.value)} />
        </label>
        <label>
          Booqable link
          <input value={value.booqableLink} onChange={(e) => set('booqableLink', e.target.value)} />
        </label>
        <label>
          Manual link
          <input value={value.manualLink} onChange={(e) => set('manualLink', e.target.value)} />
        </label>
        <label>
          Inspection link
          <input value={value.inspectionLink} onChange={(e) => set('inspectionLink', e.target.value)} />
        </label>
        <label>
          Status
          <select value={value.status} onChange={(e) => set('status', e.target.value as ProductStatus)}>
            {statuses.map((status) => (
              <option key={status.value} value={status.value}>
                {status.label}
              </option>
            ))}
          </select>
        </label>
        <label className="span-2">
          Description
          <textarea value={value.description} onChange={(e) => set('description', e.target.value)} />
        </label>
      </div>
      <button className="btn-primary" onClick={onSubmit}>
        {submitLabel}
      </button>
    </section>
  )
}

function renderLink(label: string, href?: string | null) {
  return (
    <tr>
      <th>{label}</th>
      <td>
        {href ? (
          <a href={href} target="_blank" rel="noreferrer">
            {href}
          </a>
        ) : (
          '—'
        )}
      </td>
    </tr>
  )
}

export function App() {
  const [products, setProducts] = useState<Product[]>([])
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [createInput, setCreateInput] = useState<ProductInput>(emptyInput)
  const [editInput, setEditInput] = useState<ProductInput>(emptyInput)
  const [error, setError] = useState('')

  const selectedId = parseProductIdFromHash(window.location.hash)

  const loadProducts = async () => {
    const data = await gql<{ products: Product[] }>(listQuery)
    setProducts(data.products)
  }

  const loadSelectedProduct = async (id: string) => {
    const data = await gql<{ product: Product | null }>(detailQuery, { id })
    setSelectedProduct(data.product)
    if (data.product) {
      setEditInput(fromProduct(data.product))
    }
  }

  const refresh = async () => {
    await loadProducts()
    if (selectedId) {
      await loadSelectedProduct(selectedId)
    }
  }

  useEffect(() => {
    const run = async () => {
      try {
        setError('')
        await refresh()
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err))
      }
    }

    run()

    const onHashChange = () => {
      run()
    }

    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [selectedId])

  const groupedProducts = useMemo(() => {
    const grouped: Record<ProductStatus, Product[]> = {
      mafo: [],
      'write-manual': [],
      'all-done': [],
    }
    for (const product of products) {
      grouped[product.status].push(product)
    }
    return grouped
  }, [products])

  const goToList = () => {
    window.location.hash = '#/'
  }

  const goToDetail = (id: string) => {
    window.location.hash = `#/products/${id}`
  }

  const createProduct = async () => {
    try {
      setError('')
      await gql(createMutation, { input: toGraphQLInput(createInput) })
      setCreateInput(emptyInput)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const updateProduct = async () => {
    if (!selectedId) return
    try {
      setError('')
      await gql(updateMutation, { id: selectedId, input: toGraphQLInput(editInput) })
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const deleteProduct = async (id: string) => {
    try {
      setError('')
      await gql(deleteMutation, { id })
      if (selectedId === id) {
        goToList()
      }
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <main className="page">
      <header className="header-row">
        <div>
          <h1>Products Manager</h1>
          <p>Manage products, workflow status, and documentation links.</p>
        </div>
        {selectedId ? (
          <button className="btn-secondary" onClick={goToList}>
            Back to list
          </button>
        ) : null}
      </header>

      {error ? <p className="error">{error}</p> : null}

      {!selectedId ? (
        <>
          <ProductForm
            title="Create product"
            submitLabel="Create product"
            value={createInput}
            onChange={setCreateInput}
            onSubmit={createProduct}
          />

          <section className="card">
            <h2>Products by status (kanban)</h2>
            <div className="kanban-grid">
              {statuses.map((status) => (
                <article key={status.value} className="kanban-column">
                  <h3>{status.label}</h3>
                  <ul>
                    {groupedProducts[status.value].map((product) => (
                      <li key={product.id}>
                        <button className="link-button" onClick={() => goToDetail(product.id)}>
                          {product.name}
                        </button>
                      </li>
                    ))}
                    {groupedProducts[status.value].length === 0 ? <li className="muted">No products</li> : null}
                  </ul>
                </article>
              ))}
            </div>
          </section>

          <section className="card">
            <h2>Products table</h2>
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Status</th>
                  <th>Description</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {products.map((product) => (
                  <tr key={product.id}>
                    <td>
                      <button className="link-button" onClick={() => goToDetail(product.id)}>
                        {product.name}
                      </button>
                    </td>
                    <td>{product.status}</td>
                    <td>{product.description || '—'}</td>
                    <td className="actions">
                      <button className="btn-secondary" onClick={() => goToDetail(product.id)}>
                        View
                      </button>
                      <button className="btn-danger" onClick={() => deleteProduct(product.id)}>
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
                {products.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="muted">
                      No products yet.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </section>
        </>
      ) : (
        <>
          <section className="card">
            <h2>Product details</h2>
            {selectedProduct ? (
              <table className="detail-table">
                <tbody>
                  <tr>
                    <th>Name</th>
                    <td>{selectedProduct.name}</td>
                  </tr>
                  <tr>
                    <th>Status</th>
                    <td>{selectedProduct.status}</td>
                  </tr>
                  <tr>
                    <th>Description</th>
                    <td>{selectedProduct.description || '—'}</td>
                  </tr>
                  {renderLink('Purchase link', selectedProduct.purchaseLink)}
                  {renderLink('Shop link', selectedProduct.shopLink)}
                  {renderLink('Booqable link', selectedProduct.booqableLink)}
                  {renderLink('Manual link', selectedProduct.manualLink)}
                  {renderLink('Inspection link', selectedProduct.inspectionLink)}
                  <tr>
                    <th>Created</th>
                    <td>{new Date(selectedProduct.createdAt).toLocaleString()}</td>
                  </tr>
                  <tr>
                    <th>Updated</th>
                    <td>{new Date(selectedProduct.updatedAt).toLocaleString()}</td>
                  </tr>
                </tbody>
              </table>
            ) : (
              <p className="muted">Product not found.</p>
            )}

            {selectedProduct ? (
              <button className="btn-danger" onClick={() => deleteProduct(selectedProduct.id)}>
                Delete product
              </button>
            ) : null}
          </section>

          {selectedProduct ? (
            <ProductForm
              title="Edit product"
              submitLabel="Save changes"
              value={editInput}
              onChange={setEditInput}
              onSubmit={updateProduct}
            />
          ) : null}
        </>
      )}
    </main>
  )
}
