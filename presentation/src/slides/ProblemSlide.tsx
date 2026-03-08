import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Card } from '../components/Card'

const ACCENT_CYCLE = ['cyan', 'green', 'purple', 'amber', 'cyan', 'green'] as const

export function ProblemSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="mb-12 text-center">
        <h2 className="text-4xl font-bold text-fg mb-3">{t.problem.title}</h2>
        <p className="text-muted text-lg max-w-2xl mx-auto">{t.problem.subtitle}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {t.problem.items.map((item, index) => {
          const accent = ACCENT_CYCLE[index % ACCENT_CYCLE.length]
          const number = String(index + 1).padStart(2, '0')

          return (
            <Card key={index} accent={accent} glow>
              <span className={`text-${accent} font-mono text-sm opacity-60`}>
                {number}
              </span>
              <h3 className="font-semibold text-fg mt-2 mb-1">{item.title}</h3>
              <p className="text-muted text-sm leading-relaxed">{item.description}</p>
            </Card>
          )
        })}
      </div>
    </SlideShell>
  )
}
