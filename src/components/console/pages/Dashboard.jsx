import { useNavigate } from 'react-router-dom'
import { clearSession, getUser } from '../../../api/auth'

function Dashboard() {
  const navigate = useNavigate()
  const user = getUser()

  const handleLogout = () => {
    clearSession()
    navigate('/login', { replace: true })
  }

  return (
    <div style={{
      padding: '40px 24px',
      fontFamily: "'Plus Jakarta Sans', sans-serif",
      display: 'flex',
      flexDirection: 'column',
      gap: '12px',
    }}>
      <h1 style={{ color: 'var(--color-forest)', margin: 0, fontSize: '1.6rem', fontWeight: 700 }}>
        Welcome back, {user?.login_id || 'User'}
      </h1>
      <p style={{ color: 'var(--color-muted)', margin: 0, fontSize: '0.95rem' }}>
        Sanchay IMS — your inventory at a glance.
      </p>
      <button
        onClick={handleLogout}
        style={{
          marginTop: '8px',
          padding: '10px 20px',
          background: 'var(--color-forest)',
          color: '#fafaf7',
          border: 'none',
          borderRadius: 'var(--radius-md)',
          fontSize: '14px',
          fontWeight: 600,
          cursor: 'pointer',
          fontFamily: 'inherit',
          alignSelf: 'flex-start',
        }}
      >
        Log Out
      </button>
    </div>
  )
}

export default Dashboard
