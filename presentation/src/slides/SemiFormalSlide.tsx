import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Card } from '../components/Card'
import { Badge } from '../components/Badge'

const ACCENT_CYCLE = ['cyan', 'green', 'purple', 'amber'] as const

export function SemiFormalSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="mb-10 text-center">
        <div className="flex items-center justify-center gap-3 mb-3">
          <h2 className="text-4xl font-bold text-fg">{t.semiFormal.title}</h2>
          <Badge color="green">v1.1</Badge>
        </div>
        <p className="text-muted text-lg max-w-2xl mx-auto">{t.semiFormal.subtitle}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {t.semiFormal.protocols.map((protocol, index) => {
          const accent = ACCENT_CYCLE[index % ACCENT_CYCLE.length]

          return (
            <Card key={index} accent={accent} glow className="relative overflow-hidden">
              {/* Subtle top highlight */}
              <div
                className="absolute top-0 left-0 right-0 h-px"
                style={{
                  background: `linear-gradient(90deg, transparent, var(--color-${accent}), transparent)`,
                }}
              />

              {/* Header */}
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-lg text-fg">{protocol.name}</h3>
                <span className="font-mono text-xs bg-surface px-2 py-1 rounded text-muted border border-edge">
                  {protocol.phase}
                </span>
              </div>

              {/* Description */}
              <p className="text-muted text-sm mb-4 leading-relaxed">{protocol.description}</p>

              {/* Steps - terminal style */}
              <div className="bg-panel rounded-lg p-3 border border-edge">
                <div className="space-y-1.5">
                  {protocol.steps.map((step, stepIndex) => (
                    <div key={stepIndex} className="flex items-start gap-2">
                      <span className="text-green font-mono text-sm shrink-0 mt-px">
                        {String(stepIndex + 1).padStart(2, '0')}
                      </span>
                      <span className="font-mono text-sm text-fg/80 leading-snug">
                        {step}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </Card>
          )
        })}
      </div>
    </SlideShell>
  )
}
