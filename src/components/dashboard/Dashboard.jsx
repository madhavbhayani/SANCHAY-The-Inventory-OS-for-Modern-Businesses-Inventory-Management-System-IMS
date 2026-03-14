import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { clearSession, getToken, getUser } from '../../api/auth'

function Dashboard() {
  const navigate = useNavigate()

  useEffect(() => {
    if (!getToken()) {
      navigate('/login', { replace: true })
    }
  }, [navigate])

  const user = getUser()

  const handleLogout = () => {
    clearSession()
    navigate('/login', { replace: true })
  }

  return (
    <div style={{
      minHeight: '100vh',
      background: '#fafaf7',
      fontFamily: "'Plus Jakarta Sans', sans-serif",
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      gap: '16px',
    }}>
      <h1 style={{ color: '#1a3a2a', margin: 0 }}>
        Welcome, {user?.login_id || 'User'}!
      </h1>
      <p style={{ color: '#7a7060', margin: 0 }}>
        Sanchay IMS Dashboard — coming soon.
      </p>
      <button
        onClick={handleLogout}
        style={{
          marginTop: '8px',
          padding: '10px 24px',
          background: '#1a3a2a',
          color: '#fafaf7',
          border: 'none',
          borderRadius: '10px',
          fontSize: '14px',
          fontWeight: 600,
          cursor: 'pointer',
          fontFamily: 'inherit',
        }}
      >
        Log Out
      </button>
    </div>
  )
}

export default Dashboard
