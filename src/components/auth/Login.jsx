import { useState } from 'react'
import { Link } from 'react-router-dom'
import SanchayLogo from '../landing/SanchayLogo'
import '../../styles/auth/auth.css'

function Login() {
  const [formData, setFormData] = useState({ loginId: '', password: '' })

  const handleChange = (e) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
  }

  const handleSubmit = (e) => {
    e.preventDefault()
    // TODO: connect to auth logic
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

            <button type="submit" className="auth-submit-btn">
              Log In
            </button>
          </form>

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
