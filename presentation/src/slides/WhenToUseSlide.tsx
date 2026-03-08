import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

const spectrumStyles: ReadonlyArray<{
  readonly border: string
  readonly bg: string
  readonly accent: string
  readonly width: string
  readonly glow: string
}> = [
  {
    border: 'border-l border-edge/40',
    bg: 'bg-surface/30',
    accent: 'text-muted',
    width: 'max-w-md',
    glow: '',
  },
  {
    border: 'border-l-2 border-edge',
    bg: 'bg-surface/50',
    accent: 'text-fg',
    width: 'max-w-lg',
    glow: '',
  },
  {
    border: 'border-l-2 border-cyan',
    bg: 'bg-surface/70',
    accent: 'text-cyan',
    width: 'max-w-xl',
    glow: '',
  },
  {
    border: 'border-l-3 border-green',
    bg: 'bg-surface',
    accent: 'text-green',
    width: 'max-w-2xl',
    glow: '',
  },
  {
    border: 'border-l-4 border-purple',
    bg: 'bg-surface',
    accent: 'text-purple',
    width: 'max-w-full',
    glow: 'glow-purple',
  },
]

export function WhenToUseSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="space-y-3">
          <h2 className="text-4xl font-black text-fg">{t.whenToUse.title}</h2>
          <p className="text-xl text-muted font-medium">{t.whenToUse.subtitle}</p>
        </div>

        {/* Vertical spectrum */}
        <div className="space-y-3">
          {t.whenToUse.spectrum.map((item, index) => {
            const style = spectrumStyles[index]
            const s = style ?? spectrumStyles[0]!

            return (
              <div
                key={item.level}
                className={`${s.width} ${s.border} ${s.bg} ${s.glow} rounded-r-lg transition-all`}
                style={{ paddingLeft: `${(index + 1) * 4 + 12}px` }}
              >
                <div className="py-3 pr-4 flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
                  <span className={`font-bold text-sm ${s.accent} shrink-0 min-w-[100px]`}>
                    {item.level}
                  </span>
                  <span className="text-muted text-sm flex-1">
                    {item.description}
                  </span>
                  <span className="font-mono text-cyan text-xs shrink-0">
                    {item.approach}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </SlideShell>
  )
}
