import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Badge } from '../components/Badge'

export function AdvancedV11Slide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="space-y-3">
          <h2 className="text-4xl font-black text-fg">
            {t.advancedV11.title}{' '}
            <Badge color="green">v1.1</Badge>
          </h2>
          <p className="text-xl text-cyan font-medium">{t.advancedV11.subtitle}</p>
        </div>

        {/* Two-column layout */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Left column — EET */}
          <div className="space-y-5">
            <h3 className="text-xl font-semibold text-amber">
              {t.advancedV11.eet.title}
            </h3>
            <p className="text-muted text-sm leading-relaxed">
              {t.advancedV11.eet.description}
            </p>

            {/* Vertical step flow */}
            <div className="relative pl-6">
              {/* Vertical connecting line */}
              <div className="absolute left-[11px] top-2 bottom-2 border-l-2 border-amber/40" />

              <div className="space-y-4">
                {t.advancedV11.eet.steps.map((step, index) => (
                  <div key={step} className="relative flex items-start gap-3">
                    {/* Numbered dot */}
                    <div className="absolute -left-6 top-0.5 w-[22px] h-[22px] rounded-full bg-amber/20 border border-amber flex items-center justify-center shrink-0 z-10">
                      <span className="text-amber font-mono text-xs font-bold">
                        {index + 1}
                      </span>
                    </div>
                    <span className="font-mono text-sm text-fg leading-relaxed pl-1">
                      {step}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Right column — Dynamic Rubric */}
          <div className="space-y-5">
            <h3 className="text-xl font-semibold text-purple">
              {t.advancedV11.rubric.title}
            </h3>
            <p className="text-muted text-sm leading-relaxed">
              {t.advancedV11.rubric.description}
            </p>

            {/* Table */}
            <div className="bg-surface rounded-lg overflow-hidden border border-edge">
              {/* Header row */}
              <div className="grid grid-cols-3 bg-panel text-xs uppercase text-muted tracking-wide">
                <div className="px-4 py-3 font-semibold">Criterion</div>
                <div className="px-4 py-3 font-semibold">Source</div>
                <div className="px-4 py-3 font-semibold text-right">Weight</div>
              </div>

              {/* Data rows */}
              {t.advancedV11.rubric.rows.map((row, index) => (
                <div
                  key={row.criterion}
                  className={`grid grid-cols-3 border-t border-edge/50 ${
                    index % 2 === 1 ? 'bg-surface/50' : ''
                  }`}
                >
                  <div className="px-4 py-3 text-sm text-fg font-medium">
                    {row.criterion}
                  </div>
                  <div className="px-4 py-3 text-sm text-muted">
                    {row.source}
                  </div>
                  <div className="px-4 py-3 text-sm text-right">
                    <span
                      className={`font-mono font-bold ${
                        row.weight === 'CRITICAL' ? 'text-amber' : 'text-cyan'
                      }`}
                    >
                      {row.weight}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </SlideShell>
  )
}
