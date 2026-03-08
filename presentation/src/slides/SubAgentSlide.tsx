import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

export function SubAgentSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="space-y-3">
          <h2 className="text-4xl font-black text-fg">{t.subAgent.title}</h2>
          <p className="text-xl text-cyan font-medium">{t.subAgent.subtitle}</p>
        </div>

        {/* Table */}
        <div className="bg-surface rounded-xl overflow-hidden border border-edge">
          {/* Header row */}
          <div className="grid grid-cols-3 bg-panel">
            <div className="px-6 py-4 text-xs uppercase text-muted tracking-wide font-semibold">
              Phase
            </div>
            <div className="px-6 py-4 text-xs uppercase text-muted tracking-wide font-semibold">
              Model
            </div>
            <div className="px-6 py-4 text-xs uppercase text-muted tracking-wide font-semibold">
              Reasoning
            </div>
          </div>

          {/* Data rows */}
          {t.subAgent.rows.map((row, index) => (
            <div
              key={row.phase}
              className={`grid grid-cols-3 border-t border-edge/50 ${
                index % 2 === 1 ? 'bg-surface/50' : 'bg-surface'
              }`}
            >
              <div className="px-6 py-4 font-mono text-sm text-fg">
                {row.phase}
              </div>
              <div className="px-6 py-4">
                <span
                  className={`font-bold text-sm ${
                    row.model === 'Opus' ? 'text-purple' : 'text-cyan'
                  }`}
                >
                  {row.model}
                </span>
              </div>
              <div className="px-6 py-4 text-sm text-muted">
                {row.reason}
              </div>
            </div>
          ))}
        </div>

        {/* Cost note */}
        <div className="bg-green/10 border border-green/30 text-green rounded-lg p-4 text-center text-sm font-medium">
          {t.subAgent.costNote}
        </div>
      </div>
    </SlideShell>
  )
}
