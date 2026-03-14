import sanchayLogo from '../../assets/sanchay-logo.png'

function SanchayLogo({
  size = 36,
  wordmarkColor = '#1A3A2A',
  wordmarkSize = 20,
  className = '',
}) {
  return (
    <div className={`sanchay-logo ${className}`.trim()}>
      <img
        className="sanchay-logo-mark"
        src={sanchayLogo}
        width={size}
        height={size}
        alt=""
        aria-hidden="true"
      />
      <span
        className="sanchay-logo-wordmark"
        style={{ color: wordmarkColor, fontSize: `${wordmarkSize}px` }}
      >
        Sanchay
      </span>
    </div>
  )
}

export default SanchayLogo
