import { useState } from 'react'
import { Link } from 'react-router-dom'
import SanchayLogo from '../landing/SanchayLogo'
import '../../styles/auth/auth.css'

const PASSWORD_RULES = [
  { id: 'length', label: 'At least 8 characters', test: (p) => p.length >= 8 },
  { id: 'lower', label: 'One lowercase letter (a–z)', test: (p) => /[a-z]/.test(p) },
  { id: 'upper', label: 'One uppercase letter (A–Z)', test: (p) => /[A-Z]/.test(p) },
  {
    id: 'special',
    label: 'One special character (!@#$…)',
    test: (p) => /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?`~]/.test(p),
  },
]

function Signup() {
  const [formData, setFormData] = useState({
    loginId: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [errors, setErrors] = useState({})
  const [passwordFocused, setPasswordFocused] = useState(false)

  const handleChange = (e) => {
    const { name, value } = e.target
    setFormData((prev) => ({ ...prev, [name]: value }))
    if (errors[name]) {
      setErrors((prev) => ({ ...prev, [name]: '' }))
    }
  }

  const validate = () => {
    const newErrors = {}

    if (!formData.loginId.trim()) {
      newErrors.loginId = 'Login ID is required'
    }

    if (!formData.email.trim()) {
      newErrors.email = 'Email address is required'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Enter a valid email address'
    }

    const unmet = PASSWORD_RULES.filter((rule) => !rule.test(formData.password))
    if (unmet.length > 0) {
      newErrors.password = 'Password does not meet all requirements'
    }

    if (!formData.confirmPassword) {
      newErrors.confirmPassword = 'Please confirm your password'
    } else if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = 'Passwords do not match'
    }

    return newErrors
  }

  const handleSubmit = (e) => {
    e.preventDefault()
    const newErrors = validate()
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }
    // TODO: connect to auth logic
  }

  const showRules = passwordFocused || formData.password.length > 0

  return (
    <div className="auth-page">
      {/* ── Left poster ── */}
      <div className="auth-poster">
        <div className="auth-poster-inner">
          <SanchayLogo size={52} wordmarkColor="#f5f0e8" wordmarkSize={26} />

          <h1 className="auth-poster-heading">
            Start tracking.
            <br />
            Stop guessing.
          </h1>

          <p className="auth-poster-sub">
            Join warehouses across India managing inventory the smart way with
            Sanchay IMS — free to get started.
          </p>

          <ul className="auth-poster-feats">
            <li>SKU search & Smart filters</li>
            <li>Set up your warehouse in under 5 minutes</li>
            <li>Real-time inventory updates</li>
          </ul>
        </div>
      </div>

      {/* ── Right form panel ── */}
      <div className="auth-form-panel">
        <div className="auth-form-box">
          <h2 className="auth-form-title">Create account</h2>
          <p className="auth-form-subtitle">Join Sanchay IMS — it&apos;s free to start</p>

          <form className="auth-form" onSubmit={handleSubmit} noValidate>
            {/* Login ID */}
            <div className="auth-field">
              <label htmlFor="loginId" className="auth-label">
                Login ID
              </label>
              <input
                type="text"
                id="loginId"
                name="loginId"
                className={`auth-input${errors.loginId ? ' auth-input--error' : ''}`}
                placeholder="Choose a unique login ID"
                autoComplete="username"
                value={formData.loginId}
                onChange={handleChange}
              />
              {errors.loginId && <p className="auth-field-error">{errors.loginId}</p>}
            </div>

            {/* Email */}
            <div className="auth-field">
              <label htmlFor="email" className="auth-label">
                Email Address
              </label>
              <input
                type="email"
                id="email"
                name="email"
                className={`auth-input${errors.email ? ' auth-input--error' : ''}`}
                placeholder="Enter your email address"
                autoComplete="email"
                value={formData.email}
                onChange={handleChange}
              />
              {errors.email && <p className="auth-field-error">{errors.email}</p>}
            </div>

            {/* Password */}
            <div className="auth-field">
              <label htmlFor="password" className="auth-label">
                Password
              </label>
              <input
                type="password"
                id="password"
                name="password"
                className={`auth-input${errors.password ? ' auth-input--error' : ''}`}
                placeholder="Create a strong password"
                autoComplete="new-password"
                value={formData.password}
                onChange={handleChange}
                onFocus={() => setPasswordFocused(true)}
                onBlur={() => setPasswordFocused(false)}
              />
              {showRules && (
                <ul className="auth-password-rules">
                  {PASSWORD_RULES.map((rule) => (
                    <li
                      key={rule.id}
                      className={`auth-password-rule${rule.test(formData.password) ? ' is-met' : ''}`}
                    >
                      <span className="rule-dot" aria-hidden="true" />
                      {rule.label}
                    </li>
                  ))}
                </ul>
              )}
              {errors.password && !showRules && (
                <p className="auth-field-error">{errors.password}</p>
              )}
            </div>

            {/* Confirm Password */}
            <div className="auth-field">
              <label htmlFor="confirmPassword" className="auth-label">
                Re-enter Password
              </label>
              <input
                type="password"
                id="confirmPassword"
                name="confirmPassword"
                className={`auth-input${errors.confirmPassword ? ' auth-input--error' : ''}`}
                placeholder="Confirm your password"
                autoComplete="new-password"
                value={formData.confirmPassword}
                onChange={handleChange}
              />
              {errors.confirmPassword && (
                <p className="auth-field-error">{errors.confirmPassword}</p>
              )}
            </div>

            <button type="submit" className="auth-submit-btn">
              Create Account
            </button>
          </form>

          <hr className="auth-divider" />

          <p className="auth-switch">
            Already have an account?{' '}
            <Link to="/login" className="auth-link">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}

export default Signup
