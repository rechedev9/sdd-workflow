import type { ReactNode } from 'react'

type Accent = 'cyan' | 'green' | 'purple' | 'amber'

type CardProps = {
  readonly children: ReactNode
  readonly className?: string
  readonly accent?: Accent
  readonly glow?: boolean
}

const accentBorderColors: Record<Accent, string> = {
  cyan: 'border-l-cyan',
  green: 'border-l-green',
  purple: 'border-l-purple',
  amber: 'border-l-amber',
}

export function Card({ children, className = '', accent, glow = false }: CardProps): ReactNode {
  const accentBorder = accent ? `border-l-2 ${accentBorderColors[accent]}` : ''
  const glowClass = glow ? 'card-glow' : ''

  return (
    <div
      className={`bg-surface border border-edge rounded-xl p-6 ${accentBorder} ${glowClass} ${className}`}
    >
      {children}
    </div>
  )
}
