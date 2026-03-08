import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

export function ComparisonSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-10">
        {/* Header */}
        <div className="space-y-3">
          <h2 className="text-4xl font-black text-fg">
            {t.comparison.title}
          </h2>
          <p className="text-xl text-muted font-medium">{t.comparison.subtitle}</p>
        </div>

        {/* Two-column comparison */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Left — v1.0 (Before) */}
          <div className="space-y-4 opacity-75">
            <h3 className="text-2xl text-muted font-bold">
              {t.comparison.before.title}
            </h3>
            <div className="space-y-2">
              {t.comparison.before.items.map((item) => (
                <div
                  key={item}
                  className="bg-surface/50 p-3 rounded-lg flex items-start gap-3"
                >
                  <span className="text-red-400 shrink-0 font-bold mt-0.5">&#10007;</span>
                  <span className="text-muted text-sm">{item}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Right — v1.1 (After) */}
          <div className="space-y-4">
            <h3 className="text-2xl text-green font-bold glow-green">
              {t.comparison.after.title}
            </h3>
            <div className="space-y-2">
              {t.comparison.after.items.map((item) => (
                <div
                  key={item}
                  className="bg-surface p-3 rounded-lg border-l-2 border-green flex items-start gap-3"
                >
                  <span className="text-green shrink-0 font-bold mt-0.5">&#10003;</span>
                  <span className="text-fg text-sm">{item}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </SlideShell>
  )
}
