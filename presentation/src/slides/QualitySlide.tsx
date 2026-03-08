import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

export function QualitySlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="text-center space-y-2">
          <h2 className="text-4xl font-black text-fg">{t.quality.title}</h2>
          <p className="text-lg text-muted">{t.quality.subtitle}</p>
        </div>

        {/* Two columns */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Left: Automatic tracking */}
          <div className="space-y-4">
            <h3 className="font-mono text-cyan text-base font-semibold">
              {t.quality.timelineTitle}
            </h3>
            <p className="text-muted text-xs leading-relaxed">
              {t.quality.timelineDescription}
            </p>
            <div className="space-y-2">
              {t.quality.fields.map((field) => (
                <div
                  key={field.name}
                  className="flex items-center gap-3 bg-surface border border-edge rounded-lg px-3 py-2.5"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-cyan shrink-0" />
                  <span className="text-fg text-sm">{field.description}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Right: Dashboard */}
          <div className="space-y-4">
            <h3 className="font-mono text-green text-base font-semibold">
              {t.quality.analyticsTitle}
            </h3>
            <div className="space-y-2">
              {t.quality.analyticsList.map((item) => (
                <div
                  key={item}
                  className="flex items-center gap-3 bg-surface border border-edge rounded-lg px-3 py-2.5"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-green shrink-0" />
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
