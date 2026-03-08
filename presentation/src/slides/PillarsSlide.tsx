import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Card } from '../components/Card'
import { Badge } from '../components/Badge'

const ACCENT_CYCLE = ['cyan', 'green', 'purple', 'amber', 'cyan'] as const

export function PillarsSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="mb-12 text-center">
        <h2 className="text-4xl font-bold text-fg mb-3">{t.pillars.title}</h2>
        <p className="text-muted text-lg max-w-2xl mx-auto">{t.pillars.subtitle}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {t.pillars.items.map((item, index) => {
          const accent = ACCENT_CYCLE[index % ACCENT_CYCLE.length]
          const isLastRow = index >= 3 && t.pillars.items.length === 5

          return (
            <Card
              key={index}
              accent={accent}
              glow
              className={isLastRow && index === 3 ? 'lg:col-start-1 lg:col-end-2' : isLastRow && index === 4 ? 'lg:col-start-2 lg:col-end-4' : ''}
            >
              <div className="flex items-center gap-2 mb-2">
                <h3 className="font-semibold text-lg text-fg">{item.title}</h3>
                {'isV11' in item && item.isV11 ? <Badge color="green">v1.1</Badge> : null}
              </div>
              <p className="text-muted leading-relaxed">{item.description}</p>
              {item.detail ? (
                <p className="text-sm italic text-muted/70 mt-2">{item.detail}</p>
              ) : null}
            </Card>
          )
        })}
      </div>
    </SlideShell>
  )
}
