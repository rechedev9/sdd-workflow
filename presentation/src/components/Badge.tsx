import type { ReactNode } from 'react'

type BadgeColor = 'cyan' | 'green' | 'purple' | 'amber'

type BadgeProps = {
  readonly children: ReactNode
  readonly color?: BadgeColor
}

const badgeStyles: Record<BadgeColor, string> = {
  cyan: 'bg-cyan/10 text-cyan',
  green: 'bg-green/10 text-green',
  purple: 'bg-purple/10 text-purple',
  amber: 'bg-amber/10 text-amber',
}

export function Badge({ children, color = 'cyan' }: BadgeProps): ReactNode {
  const colorClasses = badgeStyles[color]
  const pulseClass = color === 'green' ? 'v11-pulse' : ''

  return (
    <span
      className={`inline-block font-mono text-xs rounded-full px-3 py-1 ${colorClasses} ${pulseClass}`}
    >
      {children}
    </span>
  )
}
