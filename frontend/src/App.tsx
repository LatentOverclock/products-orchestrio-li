import { FormEvent, useEffect, useMemo, useState } from 'react'
import { clearAuthToken, getAuthToken, gql, setAuthToken } from './api'

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

type User = {
  id: string
  email: string
  createdAt: string
  updatedAt: string
}

type UserDraft = {
  email: string
  password: string
}

type RouteState =
  | { kind: 'table' }
  | { kind: 'kanban' }
  | { kind: 'create' }
  | { kind: 'users' }
  | { kind: 'product'; id: string }

type TablePropertyKey =
  | 'name'
  | 'purchaseLink'
  | 'shopLink'
  | 'booqableLink'
  | 'manualLink'
  | 'inspectionLink'
  | 'status'

type SortDirection = 'asc' | 'desc'

const statuses: Array<{ value: ProductStatus; label: string }> = [
  { value: 'mafo', label: 'mafo' },
  { value: 'write-manual', label: 'write-manual' },
  { value: 'all-done', label: 'all-done' },
]

const tablePropertyOrder: TablePropertyKey[] = [
  'name',
  'purchaseLink',
  'shopLink',
  'booqableLink',
  'manualLink',
  'inspectionLink',
  'status',
]

const tablePropertyLabels: Record<TablePropertyKey, string> = {
  name: 'Name',
  purchaseLink: 'Purchase link',
  shopLink: 'Shop link',
  booqableLink: 'Booqable link',
  manualLink: 'Manual link',
  inspectionLink: 'Inspection link',
  status: 'Status',
}

const initialTableFilters: Record<TablePropertyKey, string> = {
  name: '',
  purchaseLink: '',
  shopLink: '',
  booqableLink: '',
  manualLink: '',
  inspectionLink: '',
  status: '',
}

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

const listProductsQuery = `
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

const productByIdQuery = `
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

const createProductMutation = `
mutation CreateProduct($input: ProductInput!) {
  createProduct(input: $input) { id }
}`

const updateProductMutation = `
mutation UpdateProduct($id: ID!, $input: ProductInput!) {
  updateProduct(id: $id, input: $input) { id }
}`

const deleteProductMutation = `
mutation DeleteProduct($id: ID!) {
  deleteProduct(id: $id)
}`

const listUsersQuery = `
query ListUsers {
  users {
    id
    email
    createdAt
    updatedAt
  }
}`

const updateUserMutation = `
mutation UpdateUser($id: ID!, $input: UpdateUserInput!) {
  updateUser(id: $id, input: $input) {
    id
    email
    updatedAt
  }
}`

const deleteUserMutation = `
mutation DeleteUser($id: ID!) {
  deleteUser(id: $id)
}`

function parseRoute(hash: string): RouteState {
  if (hash === '#/kanban') {
    return { kind: 'kanban' }
  }

  if (hash === '#/products/new') {
    return { kind: 'create' }
  }

  if (hash === '#/users') {
    return { kind: 'users' }
  }

  const productMatch = hash.match(/^#\/products\/(\d+)$/)
  if (productMatch) {
    return { kind: 'product', id: productMatch[1] }
  }

  return { kind: 'table' }
}

function productTableValue(product: Product, key: TablePropertyKey): string {
  if (key === 'name') return product.name
  if (key === 'purchaseLink') return product.purchaseLink ?? ''
  if (key === 'shopLink') return product.shopLink ?? ''
  if (key === 'booqableLink') return product.booqableLink ?? ''
  if (key === 'manualLink') return product.manualLink ?? ''
  if (key === 'inspectionLink') return product.inspectionLink ?? ''
  return product.status
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

function renderTableLink(href?: string | null) {
  if (!href) {
    return <span className="muted">—</span>
  }

  return (
    <a className="table-link" href={href} target="_blank" rel="noreferrer">
      {href}
    </a>
  )
}

function isUnauthorizedError(message: string): boolean {
  return message.toLowerCase().includes('unauthorized') || message.toLowerCase().includes('invalid token')
}

export function App() {
  const [route, setRoute] = useState<RouteState>(() => parseRoute(window.location.hash))
  const [authTokenState, setAuthTokenState] = useState<string | null>(() => getAuthToken())
  const [currentUserEmail, setCurrentUserEmail] = useState('')

  const [loginEmail, setLoginEmail] = useState('admin@products.local')
  const [loginPassword, setLoginPassword] = useState('admin123')
  const [loginError, setLoginError] = useState('')
  const [isLoggingIn, setIsLoggingIn] = useState(false)

  const [products, setProducts] = useState<Product[]>([])
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [createInput, setCreateInput] = useState<ProductInput>(emptyInput)
  const [editInput, setEditInput] = useState<ProductInput>(emptyInput)
  const [error, setError] = useState('')

  const [users, setUsers] = useState<User[]>([])
  const [userDrafts, setUserDrafts] = useState<Record<string, UserDraft>>({})
  const [usersError, setUsersError] = useState('')

  const [tableFilters, setTableFilters] = useState<Record<TablePropertyKey, string>>(initialTableFilters)
  const [tableSortKey, setTableSortKey] = useState<TablePropertyKey>('name')
  const [tableSortDirection, setTableSortDirection] = useState<SortDirection>('asc')

  const selectedId = route.kind === 'product' ? route.id : null

  const logout = (reason?: string) => {
    clearAuthToken()
    setAuthTokenState(null)
    setCurrentUserEmail('')
    setProducts([])
    setSelectedProduct(null)
    setUsers([])
    setUserDrafts({})
    if (reason) {
      setLoginError(reason)
    }
  }

  const handleOperationalError = (err: unknown, setTargetError: (message: string) => void) => {
    const message = err instanceof Error ? err.message : String(err)
    if (isUnauthorizedError(message)) {
      logout('Session expired. Please login again.')
      return
    }
    setTargetError(message)
  }

  useEffect(() => {
    const onHashChange = () => {
      setRoute(parseRoute(window.location.hash))
    }

    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])

  const loadProducts = async () => {
    const data = await gql<{ products: Product[] }>(listProductsQuery)
    setProducts(data.products)
  }

  const loadSelectedProduct = async (id: string) => {
    const data = await gql<{ product: Product | null }>(productByIdQuery, { id })
    setSelectedProduct(data.product)
    if (data.product) {
      setEditInput(fromProduct(data.product))
    }
  }

  const loadUsers = async () => {
    const data = await gql<{ users: User[] }>(listUsersQuery)
    setUsers(data.users)
    setUserDrafts((previous) => {
      const next: Record<string, UserDraft> = {}
      for (const user of data.users) {
        next[user.id] = {
          email: previous[user.id]?.email ?? user.email,
          password: '',
        }
      }
      return next
    })
  }

  useEffect(() => {
    if (!authTokenState) {
      return
    }

    const run = async () => {
      try {
        setError('')
        setUsersError('')

        if (route.kind === 'users') {
          await loadUsers()
          return
        }

        await loadProducts()

        if (selectedId) {
          await loadSelectedProduct(selectedId)
        } else {
          setSelectedProduct(null)
        }
      } catch (err) {
        handleOperationalError(err, route.kind === 'users' ? setUsersError : setError)
      }
    }

    run()
  }, [authTokenState, route.kind, selectedId])

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

  const filteredAndSortedProducts = useMemo(() => {
    const filtered = products.filter((product) => {
      return tablePropertyOrder.every((key) => {
        const filterValue = tableFilters[key].trim().toLowerCase()
        if (!filterValue) {
          return true
        }
        const sourceValue = productTableValue(product, key).toLowerCase()
        return sourceValue.includes(filterValue)
      })
    })

    const sorted = [...filtered].sort((a, b) => {
      const aValue = productTableValue(a, tableSortKey).toLowerCase()
      const bValue = productTableValue(b, tableSortKey).toLowerCase()
      const comparison = aValue.localeCompare(bValue, undefined, { numeric: true, sensitivity: 'base' })
      return tableSortDirection === 'asc' ? comparison : -comparison
    })

    return sorted
  }, [products, tableFilters, tableSortDirection, tableSortKey])

  const navigate = (hash: string) => {
    if (window.location.hash === hash) {
      setRoute(parseRoute(hash))
      return
    }
    window.location.hash = hash
  }

  const goToTable = () => navigate('#/table')
  const goToKanban = () => navigate('#/kanban')
  const goToCreate = () => navigate('#/products/new')
  const goToProduct = (id: string) => navigate(`#/products/${id}`)
  const goToUsers = () => navigate('#/users')

  const setTableFilter = (key: TablePropertyKey, value: string) => {
    setTableFilters((current) => ({ ...current, [key]: value }))
  }

  const toggleTableSort = (key: TablePropertyKey) => {
    if (tableSortKey === key) {
      setTableSortDirection((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }

    setTableSortKey(key)
    setTableSortDirection('asc')
  }

  const tableSortIndicator = (key: TablePropertyKey) => {
    if (tableSortKey !== key) {
      return '↕'
    }
    return tableSortDirection === 'asc' ? '↑' : '↓'
  }

  const createProduct = async () => {
    try {
      setError('')
      const data = await gql<{ createProduct: { id: string } }>(createProductMutation, {
        input: toGraphQLInput(createInput),
      })
      setCreateInput(emptyInput)
      goToProduct(data.createProduct.id)
    } catch (err) {
      handleOperationalError(err, setError)
    }
  }

  const updateProduct = async () => {
    if (!selectedId) return
    try {
      setError('')
      await gql(updateProductMutation, { id: selectedId, input: toGraphQLInput(editInput) })
      await loadProducts()
      await loadSelectedProduct(selectedId)
    } catch (err) {
      handleOperationalError(err, setError)
    }
  }

  const deleteProduct = async (id: string) => {
    try {
      setError('')
      await gql(deleteProductMutation, { id })
      if (selectedId === id) {
        goToTable()
        return
      }
      await loadProducts()
    } catch (err) {
      handleOperationalError(err, setError)
    }
  }

  const setUserDraftField = (id: string, field: keyof UserDraft, value: string) => {
    setUserDrafts((current) => ({
      ...current,
      [id]: {
        email: current[id]?.email ?? '',
        password: current[id]?.password ?? '',
        [field]: value,
      },
    }))
  }

  const saveUser = async (id: string) => {
    const draft = userDrafts[id]
    if (!draft) {
      return
    }

    const input: Record<string, string> = {
      email: draft.email,
    }
    if (draft.password.trim() !== '') {
      input.password = draft.password
    }

    try {
      setUsersError('')
      await gql(updateUserMutation, { id, input })
      await loadUsers()
    } catch (err) {
      handleOperationalError(err, setUsersError)
    }
  }

  const removeUser = async (id: string) => {
    try {
      setUsersError('')
      await gql(deleteUserMutation, { id })
      await loadUsers()
    } catch (err) {
      handleOperationalError(err, setUsersError)
    }
  }

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    try {
      setIsLoggingIn(true)
      setLoginError('')

      const response = await fetch('/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: loginEmail, password: loginPassword }),
      })

      const payload = (await response.json()) as {
        token?: string
        user?: { email?: string }
        errors?: Array<{ message: string }>
      }

      if (!response.ok || !payload.token) {
        if (payload.errors?.[0]?.message) {
          throw new Error(payload.errors[0].message)
        }
        throw new Error('Login failed')
      }

      setAuthToken(payload.token)
      setAuthTokenState(payload.token)
      setCurrentUserEmail(payload.user?.email ?? '')
      setLoginPassword('')
      navigate('#/table')
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : String(err))
    } finally {
      setIsLoggingIn(false)
    }
  }

  if (!authTokenState) {
    return (
      <main className="auth-shell">
        <section className="auth-card">
          <h1>Login required</h1>
          <p>You must login to access products and user management.</p>
          {loginError ? <p className="error">{loginError}</p> : null}
          <form className="form-grid" onSubmit={handleLogin}>
            <label>
              Email
              <input type="email" value={loginEmail} onChange={(e) => setLoginEmail(e.target.value)} />
            </label>
            <label>
              Password
              <input type="password" value={loginPassword} onChange={(e) => setLoginPassword(e.target.value)} />
            </label>
            <button className="btn-primary" disabled={isLoggingIn} type="submit">
              {isLoggingIn ? 'Logging in…' : 'Login'}
            </button>
          </form>
        </section>
      </main>
    )
  }

  return (
    <main className="page">
      <header className="header-row">
        <div>
          <h1>Products Manager</h1>
          <p>Manage products, workflow status, documentation links, and users.</p>
        </div>
        <div className="session-controls">
          <span className="muted">{currentUserEmail ? `Logged in as ${currentUserEmail}` : 'Authenticated'}</span>
          <button className="btn-secondary" onClick={() => logout('Logged out.')}>Logout</button>
        </div>
      </header>

      <nav className="view-switch">
        <button className={`btn-secondary ${route.kind === 'table' ? 'is-active' : ''}`} onClick={goToTable}>
          Table view
        </button>
        <button className={`btn-secondary ${route.kind === 'kanban' ? 'is-active' : ''}`} onClick={goToKanban}>
          Kanban view
        </button>
        <button className={`btn-secondary ${route.kind === 'create' ? 'is-active' : ''}`} onClick={goToCreate}>
          Create / edit page
        </button>
        <button className={`btn-secondary ${route.kind === 'users' ? 'is-active' : ''}`} onClick={goToUsers}>
          Users
        </button>
      </nav>

      {error ? <p className="error">{error}</p> : null}

      {route.kind === 'table' ? (
        <section className="card">
          <h2>Products table</h2>
          <table className="table">
            <thead>
              <tr>
                {tablePropertyOrder.map((key) => (
                  <th key={key}>
                    <button className="header-sort" onClick={() => toggleTableSort(key)}>
                      <span>{tablePropertyLabels[key]}</span>
                      <span className="header-sort-indicator">{tableSortIndicator(key)}</span>
                    </button>
                  </th>
                ))}
                <th>Actions</th>
              </tr>
              <tr className="filter-row">
                <th>
                  <input
                    aria-label="Filter name"
                    placeholder="Filter name"
                    value={tableFilters.name}
                    onChange={(event) => setTableFilter('name', event.target.value)}
                  />
                </th>
                <th>
                  <input
                    aria-label="Filter purchase link"
                    placeholder="Filter purchase link"
                    value={tableFilters.purchaseLink}
                    onChange={(event) => setTableFilter('purchaseLink', event.target.value)}
                  />
                </th>
                <th>
                  <input
                    aria-label="Filter shop link"
                    placeholder="Filter shop link"
                    value={tableFilters.shopLink}
                    onChange={(event) => setTableFilter('shopLink', event.target.value)}
                  />
                </th>
                <th>
                  <input
                    aria-label="Filter booqable link"
                    placeholder="Filter booqable link"
                    value={tableFilters.booqableLink}
                    onChange={(event) => setTableFilter('booqableLink', event.target.value)}
                  />
                </th>
                <th>
                  <input
                    aria-label="Filter manual link"
                    placeholder="Filter manual link"
                    value={tableFilters.manualLink}
                    onChange={(event) => setTableFilter('manualLink', event.target.value)}
                  />
                </th>
                <th>
                  <input
                    aria-label="Filter inspection link"
                    placeholder="Filter inspection link"
                    value={tableFilters.inspectionLink}
                    onChange={(event) => setTableFilter('inspectionLink', event.target.value)}
                  />
                </th>
                <th>
                  <select
                    aria-label="Filter status"
                    value={tableFilters.status}
                    onChange={(event) => setTableFilter('status', event.target.value)}
                  >
                    <option value="">All statuses</option>
                    {statuses.map((status) => (
                      <option key={status.value} value={status.value}>
                        {status.label}
                      </option>
                    ))}
                  </select>
                </th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {filteredAndSortedProducts.map((product) => (
                <tr key={product.id}>
                  <td>
                    <button className="link-button" onClick={() => goToProduct(product.id)}>
                      {product.name}
                    </button>
                  </td>
                  <td>{renderTableLink(product.purchaseLink)}</td>
                  <td>{renderTableLink(product.shopLink)}</td>
                  <td>{renderTableLink(product.booqableLink)}</td>
                  <td>{renderTableLink(product.manualLink)}</td>
                  <td>{renderTableLink(product.inspectionLink)}</td>
                  <td>{product.status}</td>
                  <td className="actions">
                    <button className="btn-secondary" onClick={() => goToProduct(product.id)}>
                      Open
                    </button>
                    <button className="btn-danger" onClick={() => deleteProduct(product.id)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
              {filteredAndSortedProducts.length === 0 ? (
                <tr>
                  <td colSpan={8} className="muted">
                    No products matching current filters.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </section>
      ) : null}

      {route.kind === 'kanban' ? (
        <section className="card">
          <h2>Products by status (kanban)</h2>
          <div className="kanban-grid">
            {statuses.map((status) => (
              <article key={status.value} className="kanban-column">
                <h3>{status.label}</h3>
                <ul>
                  {groupedProducts[status.value].map((product) => (
                    <li key={product.id}>
                      <button className="link-button" onClick={() => goToProduct(product.id)}>
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
      ) : null}

      {route.kind === 'create' ? (
        <ProductForm
          title="Create / edit page"
          submitLabel="Create product"
          value={createInput}
          onChange={setCreateInput}
          onSubmit={createProduct}
        />
      ) : null}

      {route.kind === 'users' ? (
        <section className="card">
          <h2>Users management</h2>
          <p className="muted">List, edit, and delete users.</p>
          {usersError ? <p className="error">{usersError}</p> : null}
          <table className="table">
            <thead>
              <tr>
                <th>Email</th>
                <th>New password (optional)</th>
                <th>Created</th>
                <th>Updated</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id}>
                  <td>
                    <input
                      aria-label={`User email ${user.id}`}
                      value={userDrafts[user.id]?.email ?? user.email}
                      onChange={(event) => setUserDraftField(user.id, 'email', event.target.value)}
                    />
                  </td>
                  <td>
                    <input
                      aria-label={`User password ${user.id}`}
                      type="password"
                      placeholder="Leave empty to keep current password"
                      value={userDrafts[user.id]?.password ?? ''}
                      onChange={(event) => setUserDraftField(user.id, 'password', event.target.value)}
                    />
                  </td>
                  <td>{new Date(user.createdAt).toLocaleString()}</td>
                  <td>{new Date(user.updatedAt).toLocaleString()}</td>
                  <td className="actions">
                    <button className="btn-primary" onClick={() => saveUser(user.id)}>
                      Save
                    </button>
                    <button className="btn-danger" onClick={() => removeUser(user.id)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
              {users.length === 0 ? (
                <tr>
                  <td colSpan={5} className="muted">
                    No users found.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </section>
      ) : null}

      {route.kind === 'product' ?
        selectedProduct ? (
          <>
            <ProductForm
              title="View / edit page"
              submitLabel="Save changes"
              value={editInput}
              onChange={setEditInput}
              onSubmit={updateProduct}
            />

            <section className="card">
              <h2>Product metadata</h2>
              <table className="detail-table">
                <tbody>
                  <tr>
                    <th>ID</th>
                    <td>{selectedProduct.id}</td>
                  </tr>
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

              <button className="btn-danger" onClick={() => deleteProduct(selectedProduct.id)}>
                Delete product
              </button>
            </section>
          </>
        ) : (
          <section className="card">
            <h2>View / edit page</h2>
            <p className="muted">Product not found.</p>
          </section>
        )
      : null}
    </main>
  )
}
