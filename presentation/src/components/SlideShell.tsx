import type { ReactNode } from 'react'

type SlideShellProps = {
  readonly children: ReactNode
  readonly className?: string
}

export function SlideShell({ children, className = '' }: SlideShellProps): ReactNode {
  return (
    <div className="slide-bg min-h-screen w-full flex items-center justify-center p-8 md:p-16">
      <div className={`slide-enter max-w-6xl w-full ${className}`}>
        {children}
      </div>
    </div>
  )
}
