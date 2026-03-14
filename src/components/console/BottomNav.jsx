import { NavLink } from 'react-router-dom'
import {
  HiOutlineSquares2X2,
  HiOutlineClipboardDocumentList,
  HiOutlineCube,
  HiOutlineArrowsRightLeft,
  HiOutlineCog6Tooth,
} from 'react-icons/hi2'
import '../../styles/dashboard/console.css'

const NAV_ITEMS = [
  {
    to: '/dashboard',
    label: 'Dashboard',
    icon: HiOutlineSquares2X2,
  },
  {
    to: '/operations',
    label: 'Operations',
    icon: HiOutlineClipboardDocumentList,
  },
  {
    to: '/stock',
    label: 'Stock',
    icon: HiOutlineCube,
  },
  {
    to: '/move-history',
    label: 'Move History',
    icon: HiOutlineArrowsRightLeft,
  },
  {
    to: '/settings',
    label: 'Settings',
    icon: HiOutlineCog6Tooth,
  },
]

function BottomNav() {
  return (
    <nav className="bottom-nav" aria-label="Console navigation">
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          end={to !== '/operations'}
          className={({ isActive }) =>
            `bottom-nav-item${isActive ? ' is-active' : ''}`
          }
          aria-label={label}
        >
          <span className="bottom-nav-icon" aria-hidden="true">
            <Icon size={22} />
          </span>
          <span className="bottom-nav-label">{label}</span>
        </NavLink>
      ))}
    </nav>
  )
}

export default BottomNav
