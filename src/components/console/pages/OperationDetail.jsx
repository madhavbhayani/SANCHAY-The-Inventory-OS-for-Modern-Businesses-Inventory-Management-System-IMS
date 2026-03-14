import { Link, useParams } from 'react-router-dom'
import '../../../styles/dashboard/operations.css'

function OperationDetail() {
  const { operationType = 'IN', referenceNumber = '' } = useParams()

  const normalizedType = String(operationType).toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  const decodedReference = safeDecode(referenceNumber)

  return (
    <section className="operation-detail-shell">
      <article className="operation-detail-card">
        <h1>Operation Detail</h1>
        <p>
          This detail screen is intentionally pending for now. The route is active and ready for the next
          implementation step.
        </p>

        <div className="operation-detail-meta">
          <div>
            <span>Operation</span>
            <strong>{normalizedType}</strong>
          </div>
          <div>
            <span>Reference Number</span>
            <strong>{decodedReference || '--'}</strong>
          </div>
        </div>

        <div>
          <Link to="/operations" className="operations-btn secondary">
            Back to Operations
          </Link>
        </div>
      </article>
    </section>
  )
}

function safeDecode(value) {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export default OperationDetail
