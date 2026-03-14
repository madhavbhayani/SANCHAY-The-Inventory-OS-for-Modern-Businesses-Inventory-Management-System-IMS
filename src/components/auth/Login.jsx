import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import SanchayLogo from '../landing/SanchayLogo'
import { apiLogin, saveSession } from '../../api/auth'
import '../../styles/auth/auth.css'

function Login() {
  const navigate = useNavigate()
  const [formData, setFormData] = useState({ loginId: '', password: '' })
  const [apiError, setApiError] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleChange = (e) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
    if (apiError) setApiError('')
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!formData.loginId.trim() || !formData.password) return
    setIsLoading(true)
    setApiError('')
    try {
      const { token, user } = await apiLogin({
        identifier: formData.loginId.trim(),
        password: formData.password,
      })
      saveSession(token, user)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setApiError(err.message)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="auth-page">
      {/* ── Left poster ── */}
      <div className="auth-poster">
        <div className="auth-poster-inner">
          <SanchayLogo size={52} wordmarkColor="#f5f0e8" wordmarkSize={26} />

          <h1 className="auth-poster-heading">
            Your warehouse.
            <br />
            Always in sync.
          </h1>

          <p className="auth-poster-sub">
            Real-time stock tracking, smart transfers, and zero guesswork — all from
            one centralized dashboard.
          </p>

          <ul className="auth-poster-feats">
            <li>Real-time visibility across all inventory locations</li>
            <li>Role-based access for managers and staff</li>
            <li>Fast barcode-based item lookup and updates</li>
          </ul>
        </div>
      </div>

      {/* ── Right form panel ── */}
      <div className="auth-form-panel">
        <div className="auth-form-box">
          <h2 className="auth-form-title">Welcome back</h2>
          <p className="auth-form-subtitle">Sign in to your Sanchay account</p>

          <form className="auth-form" onSubmit={handleSubmit} noValidate>
            <div className="auth-field">
              <label htmlFor="loginId" className="auth-label">
                Login ID or Email
              </label>
              <input
                type="text"
                id="loginId"
                name="loginId"
                className="auth-input"
                placeholder="Enter your login ID or email"
                autoComplete="username"
                value={formData.loginId}
                onChange={handleChange}
                required
              />
            </div>

            <div className="auth-field">
              <label htmlFor="password" className="auth-label">
                Password
              </label>
              <input
                type="password"
                id="password"
                name="password"
                className="auth-input"
                placeholder="Enter your password"
                autoComplete="current-password"
                value={formData.password}
                onChange={handleChange}
                required
              />
            </div>

            <button type="submit" className="auth-submit-btn" disabled={isLoading}>
              {isLoading ? 'Signing in…' : 'Log In'}
            </button>
          </form>

          {apiError && <p className="auth-api-error">{apiError}</p>}

          <Link to="/forgot-password" className="auth-forgot-link">
            Forgot Password?
          </Link>

          <hr className="auth-divider" />

          <p className="auth-switch">
            Don&apos;t have an account?{' '}
            <Link to="/signup" className="auth-link">
              Create account
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}

export default Login
