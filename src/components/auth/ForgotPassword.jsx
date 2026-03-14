import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  apiRequestForgotPasswordOtp,
  apiResetForgotPassword,
  apiVerifyForgotPasswordOtp,
} from '../../api/auth'
import SanchayLogo from '../landing/SanchayLogo'
import '../../styles/auth/auth.css'

const PASSWORD_RULES = [
  { id: 'length', label: 'At least 8 characters', test: (p) => p.length >= 8 },
  { id: 'lower', label: 'One lowercase letter (a-z)', test: (p) => /[a-z]/.test(p) },
  { id: 'upper', label: 'One uppercase letter (A-Z)', test: (p) => /[A-Z]/.test(p) },
  {
    id: 'special',
    label: 'One special character (!@#$...)',
    test: (p) => /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?`~]/.test(p),
  },
]

function ForgotPassword() {
  const navigate = useNavigate()

  const [step, setStep] = useState('request')
  const [email, setEmail] = useState('')
  const [otp, setOtp] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  const canSubmitRequest = useMemo(() => isValidEmail(email), [email])

  const sendOtp = async (event) => {
    event.preventDefault()
    if (!canSubmitRequest) {
      setError('Enter a valid email address.')
      return
    }

    setLoading(true)
    setError('')
    setSuccessMessage('')

    try {
      await apiRequestForgotPasswordOtp({ email: email.trim().toLowerCase() })
      setStep('verify')
      setSuccessMessage('OTP sent to your email address.')
    } catch (requestError) {
      setError(requestError?.message || 'Failed to send OTP.')
    } finally {
      setLoading(false)
    }
  }

  const verifyOtp = async (event) => {
    event.preventDefault()
    const cleanedOtp = otp.replace(/\D/g, '').slice(0, 6)
    if (cleanedOtp.length !== 6) {
      setError('OTP must be exactly 6 digits.')
      return
    }

    setLoading(true)
    setError('')
    setSuccessMessage('')

    try {
      await apiVerifyForgotPasswordOtp({
        email: email.trim().toLowerCase(),
        otp: cleanedOtp,
      })
      setOtp(cleanedOtp)
      setStep('reset')
      setSuccessMessage('OTP verified. You can now set a new password.')
    } catch (verifyError) {
      setError(verifyError?.message || 'Invalid OTP.')
    } finally {
      setLoading(false)
    }
  }

  const resetPassword = async (event) => {
    event.preventDefault()

    const unmet = PASSWORD_RULES.filter((rule) => !rule.test(newPassword))
    if (unmet.length > 0) {
      setError('Password does not meet all requirements.')
      return
    }
    if (!confirmPassword) {
      setError('Please re-enter your new password.')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.')
      return
    }

    setLoading(true)
    setError('')
    setSuccessMessage('')

    try {
      await apiResetForgotPassword({
        email: email.trim().toLowerCase(),
        newPassword,
      })
      setStep('done')
      setSuccessMessage('Password changed successfully. You can now log in.')
    } catch (resetError) {
      setError(resetError?.message || 'Failed to reset password.')
    } finally {
      setLoading(false)
    }
  }

  const resendOtp = async () => {
    if (loading) return
    setLoading(true)
    setError('')
    setSuccessMessage('')

    try {
      await apiRequestForgotPasswordOtp({ email: email.trim().toLowerCase() })
      setSuccessMessage('A new OTP has been sent to your email.')
    } catch (requestError) {
      setError(requestError?.message || 'Failed to resend OTP.')
    } finally {
      setLoading(false)
    }
  }

  const renderForm = () => {
    if (step === 'request') {
      return (
        <form className="auth-form" onSubmit={sendOtp} noValidate>
          <div className="auth-field">
            <label htmlFor="forgot-email" className="auth-label">
              Email Address
            </label>
            <input
              id="forgot-email"
              type="email"
              className="auth-input"
              placeholder="Enter your registered email"
              value={email}
              onChange={(event) => {
                setEmail(event.target.value)
                if (error) setError('')
              }}
              autoComplete="email"
              required
            />
          </div>

          <button type="submit" className="auth-submit-btn" disabled={loading || !canSubmitRequest}>
            {loading ? 'Sending OTP...' : 'Send OTP'}
          </button>
        </form>
      )
    }

    if (step === 'verify') {
      return (
        <form className="auth-form" onSubmit={verifyOtp} noValidate>
          <div className="auth-field">
            <label htmlFor="forgot-otp" className="auth-label">
              Enter OTP
            </label>
            <input
              id="forgot-otp"
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              maxLength={6}
              className="auth-input"
              placeholder="6 digit OTP"
              value={otp}
              onChange={(event) => {
                setOtp(event.target.value.replace(/\D/g, '').slice(0, 6))
                if (error) setError('')
              }}
              required
            />
          </div>

          <button type="submit" className="auth-submit-btn" disabled={loading || otp.length !== 6}>
            {loading ? 'Verifying...' : 'Verify OTP'}
          </button>

          <button
            type="button"
            className="auth-inline-btn"
            onClick={resendOtp}
            disabled={loading}
          >
            Resend OTP
          </button>
        </form>
      )
    }

    if (step === 'reset') {
      return (
        <form className="auth-form" onSubmit={resetPassword} noValidate>
          <div className="auth-field">
            <label htmlFor="new-password" className="auth-label">
              New Password
            </label>
            <input
              id="new-password"
              type="password"
              className="auth-input"
              placeholder="Enter new password"
              value={newPassword}
              onChange={(event) => {
                setNewPassword(event.target.value)
                if (error) setError('')
              }}
              autoComplete="new-password"
              required
            />
            <ul className="auth-password-rules">
              {PASSWORD_RULES.map((rule) => (
                <li key={rule.id} className={`auth-password-rule${rule.test(newPassword) ? ' is-met' : ''}`}>
                  <span className="rule-dot" aria-hidden="true" />
                  {rule.label}
                </li>
              ))}
            </ul>
          </div>

          <div className="auth-field">
            <label htmlFor="confirm-new-password" className="auth-label">
              Confirm New Password
            </label>
            <input
              id="confirm-new-password"
              type="password"
              className="auth-input"
              placeholder="Re-enter new password"
              value={confirmPassword}
              onChange={(event) => {
                setConfirmPassword(event.target.value)
                if (error) setError('')
              }}
              autoComplete="new-password"
              required
            />
          </div>

          <button type="submit" className="auth-submit-btn" disabled={loading}>
            {loading ? 'Updating...' : 'Change Password'}
          </button>
        </form>
      )
    }

    return (
      <div className="auth-form">
        <button
          type="button"
          className="auth-submit-btn"
          onClick={() => navigate('/login', { replace: true })}
        >
          Back to Login
        </button>
      </div>
    )
  }

  return (
    <div className="auth-page">
      <div className="auth-poster">
        <div className="auth-poster-inner">
          <SanchayLogo size={52} wordmarkColor="#f5f0e8" wordmarkSize={26} />

          <h1 className="auth-poster-heading">
            Regain access.
            <br />
            Securely and fast.
          </h1>

          <p className="auth-poster-sub">
            Verify your registered email with OTP and set a fresh password to continue using Sanchay IMS.
          </p>

          <ul className="auth-poster-feats">
            <li>Email-based OTP verification</li>
            <li>Password reset with secure hashing</li>
            <li>Built-in password strength checks</li>
          </ul>
        </div>
      </div>

      <div className="auth-form-panel">
        <div className="auth-form-box">
          <h2 className="auth-form-title">Forgot password</h2>
          <p className="auth-form-subtitle">
            {step === 'request' && 'Enter your email to receive OTP'}
            {step === 'verify' && `Verify OTP sent to ${email}`}
            {step === 'reset' && 'Set your new password'}
            {step === 'done' && 'Your password has been reset'}
          </p>

          <div className="auth-step-strip" aria-label="Password reset steps">
            <span className={`auth-step-pill${step === 'request' ? ' is-active' : ''}`}>1. Email</span>
            <span className={`auth-step-pill${step === 'verify' ? ' is-active' : ''}`}>2. OTP</span>
            <span className={`auth-step-pill${step === 'reset' ? ' is-active' : ''}`}>3. Reset</span>
          </div>

          {renderForm()}

          {error ? <p className="auth-api-error">{error}</p> : null}
          {successMessage ? <p className="auth-api-success">{successMessage}</p> : null}

          <hr className="auth-divider" />

          <p className="auth-switch">
            Back to{' '}
            <Link to="/login" className="auth-link">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}

function isValidEmail(value) {
  const email = String(value || '').trim()
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

export default ForgotPassword
