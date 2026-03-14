import { useEffect, useMemo, useState } from 'react'
import {
  apiCreateStockCategory,
  apiCreateStockProduct,
  apiDeleteStockProduct,
  apiGetStockMeta,
  apiListStockProducts,
  apiUpdateStockProduct,
} from '../../../api/auth'
import '../../../styles/dashboard/stock.css'

const EMPTY_STOCK_LEVEL = {
  locationId: '',
  onHandQuantity: '',
  freeToUseQuantity: '',
}

const EMPTY_FORM = {
  name: '',
  cost: '',
  categoryId: '',
  description: '',
  stockLevels: [{ ...EMPTY_STOCK_LEVEL }],
}

const EMPTY_CATEGORY_FORM = {
  name: '',
  description: '',
}

function Stock() {
  const [isBootstrapping, setIsBootstrapping] = useState(true)
  const [loadingProducts, setLoadingProducts] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [feedback, setFeedback] = useState({ type: '', message: '' })
  const [categoryFeedback, setCategoryFeedback] = useState({ type: '', message: '' })

  const [categories, setCategories] = useState([])
  const [locations, setLocations] = useState([])
  const [products, setProducts] = useState([])

  const [searchInput, setSearchInput] = useState('')
  const [activeQuery, setActiveQuery] = useState('')

  const [editorOpen, setEditorOpen] = useState(false)
  const [editorMode, setEditorMode] = useState('create')
  const [editingProductId, setEditingProductId] = useState('')
  const [editingSku, setEditingSku] = useState('Auto generated')
  const [formState, setFormState] = useState(EMPTY_FORM)
  const [savingProduct, setSavingProduct] = useState(false)

  const [categoryEditorOpen, setCategoryEditorOpen] = useState(false)
  const [categoryForm, setCategoryForm] = useState(EMPTY_CATEGORY_FORM)
  const [savingCategory, setSavingCategory] = useState(false)

  const summary = useMemo(() => {
    const totalOnHand = products.reduce((total, product) => total + Number(product.on_hand_quantity || 0), 0)
    const totalFreeToUse = products.reduce(
      (total, product) => total + Number(product.free_to_use_quantity || 0),
      0,
    )
    return {
      productCount: products.length,
      totalOnHand,
      totalFreeToUse,
    }
  }, [products])

  useEffect(() => {
    bootstrap()
  }, [])

  const bootstrap = async () => {
    setIsBootstrapping(true)
    setLoadError('')

    try {
      const [meta, productsData] = await Promise.all([
        apiGetStockMeta(),
        apiListStockProducts({ query: '', limit: 120 }),
      ])

      setCategories(meta.categories || [])
      setLocations(meta.locations || [])
      setProducts(productsData.products || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to load stock data.')
    } finally {
      setIsBootstrapping(false)
    }
  }

  const fetchProducts = async (queryValue = activeQuery) => {
    setLoadingProducts(true)
    setLoadError('')

    try {
      const data = await apiListStockProducts({ query: queryValue, limit: 120 })
      setProducts(data.products || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to fetch products.')
    } finally {
      setLoadingProducts(false)
    }
  }

  const openCreateEditor = () => {
    setEditorMode('create')
    setEditorOpen(true)
    setEditingProductId('')
    setEditingSku('Auto generated')
    setFormState(cloneFormState(EMPTY_FORM))
    setFeedback({ type: '', message: '' })
  }

  const openCategoryEditor = () => {
    setCategoryEditorOpen(true)
    setCategoryFeedback({ type: '', message: '' })
  }

  const closeCategoryEditor = () => {
    setCategoryEditorOpen(false)
    setCategoryForm(EMPTY_CATEGORY_FORM)
    setCategoryFeedback({ type: '', message: '' })
  }

  const openEditEditor = (product) => {
    const sourceLevels =
      Array.isArray(product.stock_levels) && product.stock_levels.length > 0
        ? product.stock_levels.map((level) => ({
            locationId: level.location_id || '',
            onHandQuantity: String(level.on_hand_quantity ?? ''),
            freeToUseQuantity: String(level.free_to_use_quantity ?? ''),
          }))
        : [
            {
              locationId: product.location_id || '',
              onHandQuantity: String(product.on_hand_quantity ?? ''),
              freeToUseQuantity: String(product.free_to_use_quantity ?? ''),
            },
          ]

    setEditorMode('edit')
    setEditorOpen(true)
    setEditingProductId(product.id)
    setEditingSku(product.sku)
    setFormState({
      name: product.name || '',
      cost: String(product.cost ?? ''),
      categoryId: product.category_id || '',
      description: product.description || '',
      stockLevels: sourceLevels.length > 0 ? sourceLevels : [{ ...EMPTY_STOCK_LEVEL }],
    })
    setFeedback({ type: '', message: '' })
  }

  const closeEditor = () => {
    setEditorOpen(false)
    setEditorMode('create')
    setEditingProductId('')
    setEditingSku('Auto generated')
    setFormState(cloneFormState(EMPTY_FORM))
  }

  const handleFormChange = (event) => {
    const { name, value } = event.target
    setFormState((previous) => ({ ...previous, [name]: value }))
  }

  const updateStockLevel = (index, field, value) => {
    setFormState((previous) => ({
      ...previous,
      stockLevels: previous.stockLevels.map((level, levelIndex) =>
        levelIndex === index ? { ...level, [field]: value } : level,
      ),
    }))
  }

  const addStockLevelRow = () => {
    setFormState((previous) => ({
      ...previous,
      stockLevels: [...previous.stockLevels, { ...EMPTY_STOCK_LEVEL }],
    }))
  }

  const removeStockLevelRow = (index) => {
    setFormState((previous) => {
      if (previous.stockLevels.length === 1) return previous
      return {
        ...previous,
        stockLevels: previous.stockLevels.filter((_, levelIndex) => levelIndex !== index),
      }
    })
  }

  const handleCategoryFormChange = (event) => {
    const { name, value } = event.target
    setCategoryForm((previous) => ({ ...previous, [name]: value }))
  }

  const submitSearch = async (event) => {
    event.preventDefault()
    const query = searchInput.trim()
    setActiveQuery(query)
    await fetchProducts(query)
  }

  const clearSearch = async () => {
    setSearchInput('')
    setActiveQuery('')
    await fetchProducts('')
  }

  const submitProduct = async (event) => {
    event.preventDefault()
    setFeedback({ type: '', message: '' })

    const payload = toProductPayload(formState)
    if (payload.error) {
      setFeedback({ type: 'error', message: payload.error })
      return
    }

    setSavingProduct(true)
    try {
      if (editorMode === 'create') {
        const response = await apiCreateStockProduct(payload.value)
        setFeedback({
          type: 'success',
          message: `Product created successfully (${response?.product?.sku || 'SKU generated'}).`,
        })
      } else {
        await apiUpdateStockProduct(editingProductId, payload.value)
        setFeedback({ type: 'success', message: 'Product updated successfully.' })
      }

      await fetchProducts(activeQuery)
      closeEditor()
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to save product.' })
    } finally {
      setSavingProduct(false)
    }
  }

  const submitCategory = async (event) => {
    event.preventDefault()
    setCategoryFeedback({ type: '', message: '' })

    const name = categoryForm.name.trim()
    const description = categoryForm.description.trim()

    if (!name) {
      setCategoryFeedback({ type: 'error', message: 'Category name is required.' })
      return
    }

    setSavingCategory(true)
    try {
      const response = await apiCreateStockCategory({ name, description })
      const created = response?.category

      const meta = await apiGetStockMeta()
      setCategories(meta.categories || [])

      if (created?.id) {
        setFormState((previous) => ({
          ...previous,
          categoryId: previous.categoryId || created.id,
        }))
      }

      setCategoryFeedback({
        type: 'success',
        message: created?.name
          ? `Category "${created.name}" created successfully.`
          : 'Category created successfully.',
      })
      setCategoryForm(EMPTY_CATEGORY_FORM)
      setCategoryEditorOpen(false)
    } catch (error) {
      setCategoryFeedback({ type: 'error', message: error?.message || 'Failed to create category.' })
    } finally {
      setSavingCategory(false)
    }
  }

  const deleteProduct = async (product) => {
    const confirmed = window.confirm(`Delete product ${product.sku} (${product.name})?`)
    if (!confirmed) return

    setFeedback({ type: '', message: '' })
    try {
      await apiDeleteStockProduct(product.id)
      if (editingProductId === product.id) {
        closeEditor()
      }
      await fetchProducts(activeQuery)
      setFeedback({ type: 'success', message: 'Product deleted successfully.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to delete product.' })
    }
  }

  if (isBootstrapping) {
    return (
      <section className="stock-shell">
        <div className="stock-status-card">Loading stock module...</div>
      </section>
    )
  }

  if (loadError && products.length === 0) {
    return (
      <section className="stock-shell">
        <div className="stock-status-card is-error">
          <h1>Unable to load stock page</h1>
          <p>{loadError}</p>
          <button type="button" className="stock-btn primary" onClick={bootstrap}>
            Retry
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="stock-shell">
      <header className="stock-header">
        <div>
          <p className="stock-header-label">Inventory</p>
          <h1 className="stock-header-title">Stock In Hand</h1>
          <p className="stock-header-subtitle">
            Create products once and assign quantity to multiple locations.
          </p>
        </div>
        <div className="stock-header-actions">
          <button type="button" className="stock-btn secondary" onClick={bootstrap}>
            Refresh
          </button>
          <button type="button" className="stock-btn ghost" onClick={openCategoryEditor}>
            Create Category
          </button>
          <button type="button" className="stock-btn primary" onClick={openCreateEditor}>
            Create/Add Product
          </button>
        </div>
      </header>

      <section className="stock-metrics-grid">
        <article className="stock-metric-card">
          <span>Products</span>
          <strong>{summary.productCount}</strong>
        </article>
        <article className="stock-metric-card">
          <span>Total On Hand</span>
          <strong>{summary.totalOnHand}</strong>
        </article>
        <article className="stock-metric-card">
          <span>Total Free To Use</span>
          <strong>{summary.totalFreeToUse}</strong>
        </article>
      </section>

      <section className="stock-toolbar-card">
        <form className="stock-search-form" onSubmit={submitSearch}>
          <input
            type="text"
            placeholder="Search by SKU, name, category, location, description"
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
          />
          <button type="submit" className="stock-btn primary" disabled={loadingProducts}>
            {loadingProducts ? 'Searching...' : 'Search'}
          </button>
          <button type="button" className="stock-btn ghost" onClick={clearSearch}>
            Clear
          </button>
        </form>
      </section>

      {feedback.message ? (
        <p className={`stock-feedback ${feedback.type === 'error' ? 'is-error' : 'is-success'}`}>
          {feedback.message}
        </p>
      ) : null}

      {categoryFeedback.message ? (
        <p className={`stock-feedback ${categoryFeedback.type === 'error' ? 'is-error' : 'is-success'}`}>
          {categoryFeedback.message}
        </p>
      ) : null}

      {categoryEditorOpen ? (
        <section className="stock-category-card">
          <div className="stock-editor-head">
            <h2>Create Category</h2>
            <button type="button" className="stock-btn ghost" onClick={closeCategoryEditor}>
              Close
            </button>
          </div>

          <form className="stock-category-form" onSubmit={submitCategory}>
            <label>
              Category Name
              <input
                name="name"
                value={categoryForm.name}
                onChange={handleCategoryFormChange}
                placeholder="Raw Material"
              />
            </label>

            <label>
              Description
              <textarea
                name="description"
                rows={3}
                value={categoryForm.description}
                onChange={handleCategoryFormChange}
                placeholder="Describe what inventory belongs to this category"
              />
            </label>

            <div className="stock-editor-actions">
              <button type="submit" className="stock-btn primary" disabled={savingCategory}>
                {savingCategory ? 'Saving...' : 'Add Category'}
              </button>
              <button type="button" className="stock-btn ghost" onClick={closeCategoryEditor}>
                Cancel
              </button>
            </div>
          </form>
        </section>
      ) : null}

      {editorOpen ? (
        <section className="stock-editor-card">
          <div className="stock-editor-head">
            <h2>{editorMode === 'create' ? 'Add Product' : 'Update Product'}</h2>
            <button type="button" className="stock-btn ghost" onClick={closeEditor}>
              Close
            </button>
          </div>

          <p className="stock-editor-sku">
            Product ID: <strong>{editingSku}</strong>
          </p>

          <form className="stock-editor-form" onSubmit={submitProduct}>
            <label>
              Name
              <input
                name="name"
                value={formState.name}
                onChange={handleFormChange}
                placeholder="Product name"
              />
            </label>

            <label>
              Cost
              <input
                name="cost"
                type="number"
                min="0"
                step="0.01"
                value={formState.cost}
                onChange={handleFormChange}
                placeholder="0.00"
              />
            </label>

            <label className="full-width">
              Product Category
              <select name="categoryId" value={formState.categoryId} onChange={handleFormChange}>
                <option value="">Select category</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
            </label>

            <div className="full-width stock-levels-panel">
              <div className="stock-levels-header">
                <p>Stock Levels by Location</p>
                <button type="button" className="stock-btn ghost small" onClick={addStockLevelRow}>
                  Add Location Row
                </button>
              </div>

              <div className="stock-level-list">
                {formState.stockLevels.map((level, index) => (
                  <div key={`stock-level-${index}`} className="stock-level-row">
                    <select
                      value={level.locationId}
                      onChange={(event) => updateStockLevel(index, 'locationId', event.target.value)}
                    >
                      <option value="">Select location</option>
                      {locations.map((location) => (
                        <option key={location.id} value={location.id}>
                          {location.name} ({location.short_code})
                        </option>
                      ))}
                    </select>

                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={level.onHandQuantity}
                      onChange={(event) =>
                        updateStockLevel(index, 'onHandQuantity', sanitizeIntegerInput(event.target.value))
                      }
                      placeholder="On Hand"
                    />

                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={level.freeToUseQuantity}
                      onChange={(event) =>
                        updateStockLevel(index, 'freeToUseQuantity', sanitizeIntegerInput(event.target.value))
                      }
                      placeholder="Free To Use"
                    />

                    <button
                      type="button"
                      className="stock-btn small danger"
                      onClick={() => removeStockLevelRow(index)}
                      disabled={formState.stockLevels.length === 1}
                    >
                      Remove
                    </button>
                  </div>
                ))}
              </div>
            </div>

            <label className="full-width">
              Product Description
              <textarea
                name="description"
                rows={3}
                value={formState.description}
                onChange={handleFormChange}
                placeholder="Brief product notes"
              />
            </label>

            <div className="stock-editor-actions full-width">
              <button type="submit" className="stock-btn primary" disabled={savingProduct}>
                {savingProduct
                  ? 'Saving...'
                  : editorMode === 'create'
                    ? 'Add Product'
                    : 'Update Product'}
              </button>
              <button type="button" className="stock-btn ghost" onClick={closeEditor}>
                Cancel
              </button>
            </div>
          </form>
        </section>
      ) : null}

      <section className="stock-table-card">
        <div className="stock-table-head">
          <h2>Inventory In Hand</h2>
          <span>{activeQuery ? `Filtered by: "${activeQuery}"` : 'All products'}</span>
        </div>

        {loadError ? <p className="stock-feedback is-error">{loadError}</p> : null}

        {loadingProducts ? (
          <p className="stock-empty">Loading products...</p>
        ) : products.length === 0 ? (
          <p className="stock-empty">No products found. Add your first product to start tracking stock.</p>
        ) : (
          <div className="stock-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Product ID</th>
                  <th>Name</th>
                  <th>Cost</th>
                  <th>On Hand</th>
                  <th>Free To Use</th>
                  <th>Category</th>
                  <th>Stock by Location</th>
                  <th>Description</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {products.map((product) => (
                  <tr key={product.id}>
                    <td>
                      <span className="stock-sku">{product.sku}</span>
                    </td>
                    <td>{product.name}</td>
                    <td>{formatCurrency(product.cost)}</td>
                    <td>{product.on_hand_quantity}</td>
                    <td>{product.free_to_use_quantity}</td>
                    <td>{product.category_name}</td>
                    <td>
                      <div className="stock-location-list">
                        {(product.stock_levels || []).length > 0 ? (
                          (product.stock_levels || []).map((level) => (
                            <div
                              key={`${product.id}-${level.location_id}`}
                              className="stock-location-list-item"
                            >
                              <strong>
                                {level.location_name} ({level.location_short_code})
                              </strong>
                              <span>
                                On Hand {level.on_hand_quantity} | Free {level.free_to_use_quantity}
                              </span>
                              <span>{(level.warehouse_names || []).join(', ') || '--'}</span>
                            </div>
                          ))
                        ) : (
                          <div className="stock-location-list-item">
                            <strong>
                              {product.location_name} ({product.location_short_code})
                            </strong>
                            <span>
                              On Hand {product.on_hand_quantity} | Free {product.free_to_use_quantity}
                            </span>
                            <span>{(product.warehouse_names || []).join(', ') || '--'}</span>
                          </div>
                        )}
                      </div>
                    </td>
                    <td>{product.description || '--'}</td>
                    <td>
                      <div className="stock-row-actions">
                        <button
                          type="button"
                          className="stock-btn small secondary"
                          onClick={() => openEditEditor(product)}
                        >
                          Update
                        </button>
                        <button
                          type="button"
                          className="stock-btn small danger"
                          onClick={() => deleteProduct(product)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </section>
  )
}

function cloneFormState(form) {
  return {
    ...form,
    stockLevels: (form.stockLevels || []).map((level) => ({ ...level })),
  }
}

function toProductPayload(form) {
  const name = String(form.name || '').trim()
  const categoryId = String(form.categoryId || '').trim()
  const description = String(form.description || '').trim()

  const rawCost = String(form.cost || '').trim()
  if (rawCost === '') return { error: 'Cost is required.' }

  const cost = Number(rawCost)
  if (!name) return { error: 'Product name is required.' }
  if (Number.isNaN(cost) || cost < 0) return { error: 'Cost must be a valid non-negative number.' }
  if (!categoryId) return { error: 'Please select a product category.' }

  const normalizedLevels = normalizeStockLevels(form.stockLevels || [])
  if (normalizedLevels.error) {
    return { error: normalizedLevels.error }
  }

  return {
    value: {
      name,
      cost,
      category_id: categoryId,
      description,
      stock_levels: normalizedLevels.value,
    },
  }
}

function normalizeStockLevels(stockLevels) {
  if (!Array.isArray(stockLevels) || stockLevels.length === 0) {
    return { error: 'Add at least one stock location row.' }
  }

  const seenLocations = new Set()
  const levels = []

  for (const level of stockLevels) {
    const locationId = String(level.locationId || '').trim()
    const rawOnHand = String(level.onHandQuantity || '').trim()
    const rawFree = String(level.freeToUseQuantity || '').trim()

    if (!locationId && !rawOnHand && !rawFree) {
      continue
    }

    if (!locationId) return { error: 'Each stock row must include a location.' }
    if (seenLocations.has(locationId)) {
      return { error: 'Each location can only appear once in stock rows.' }
    }

    if (rawOnHand === '') return { error: 'On hand quantity is required in each stock row.' }
    if (rawFree === '') return { error: 'Free to use quantity is required in each stock row.' }

    const onHandQuantity = Number.parseInt(rawOnHand, 10)
    const freeToUseQuantity = Number.parseInt(rawFree, 10)

    if (!Number.isInteger(onHandQuantity) || onHandQuantity < 0) {
      return { error: 'On hand quantity must be a valid non-negative integer.' }
    }
    if (!Number.isInteger(freeToUseQuantity) || freeToUseQuantity < 0) {
      return { error: 'Free to use quantity must be a valid non-negative integer.' }
    }
    if (freeToUseQuantity > onHandQuantity) {
      return { error: 'Free to use quantity cannot exceed on hand quantity.' }
    }

    seenLocations.add(locationId)
    levels.push({
      location_id: locationId,
      on_hand_quantity: onHandQuantity,
      free_to_use_quantity: freeToUseQuantity,
    })
  }

  if (levels.length === 0) {
    return { error: 'Add at least one valid location with stock quantity.' }
  }

  return { value: levels }
}

function sanitizeIntegerInput(value) {
  return String(value || '').replace(/\D/g, '')
}

function formatCurrency(value) {
  const number = Number(value)
  if (Number.isNaN(number)) return '--'
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: 2,
  }).format(number)
}

export default Stock
