import type { ReactNode } from 'react'

type NavigationProps = {
  readonly current: number
  readonly total: number
  readonly onPrev: () => void
  readonly onNext: () => void
}

export function Navigation({ current, total, onPrev, onNext }: NavigationProps): ReactNode {
  const isFirst = current === 0
  const isLast = current === total - 1

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-surface/80 backdrop-blur border-t border-edge">
      <div className="max-w-6xl mx-auto flex items-center justify-between px-6 py-3">
        <button
          type="button"
          onClick={onPrev}
          disabled={isFirst}
          className={`text-fg px-3 py-1.5 rounded-lg transition-opacity ${isFirst ? 'opacity-30 cursor-not-allowed' : 'hover:bg-edge/30'}`}
          aria-label="Previous slide"
        >
          ←
        </button>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            {Array.from({ length: total }, (_, i) => (
              <span
                key={i}
                className={`block w-2 h-2 rounded-full transition-colors ${
                  i === current ? 'bg-cyan' : 'bg-edge'
                }`}
              />
            ))}
          </div>
          <span className="text-muted text-sm font-mono ml-2">
            {current + 1} / {total}
          </span>
        </div>

        <button
          type="button"
          onClick={onNext}
          disabled={isLast}
          className={`text-fg px-3 py-1.5 rounded-lg transition-opacity ${isLast ? 'opacity-30 cursor-not-allowed' : 'hover:bg-edge/30'}`}
          aria-label="Next slide"
        >
          →
        </button>
      </div>
    </nav>
  )
}
