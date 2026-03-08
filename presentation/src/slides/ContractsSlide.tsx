import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Badge } from '../components/Badge'

export function ContractsSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="space-y-3">
          <h2 className="text-4xl font-black text-fg">
            {t.contracts.title}{' '}
            <Badge color="green">v1.1</Badge>
          </h2>
          <p className="text-xl text-cyan font-medium">{t.contracts.subtitle}</p>
          <p className="text-muted text-base max-w-3xl">{t.contracts.description}</p>
        </div>

        {/* Phase cards grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {t.contracts.phases.map((phase) => (
            <div
              key={phase.name}
              className="bg-surface border border-edge rounded-xl p-6 space-y-4 hover:border-cyan/30 transition-colors"
            >
              {/* Phase name */}
              <h3 className="font-mono text-cyan text-lg font-semibold tracking-wide">
                {phase.name}
              </h3>

              {/* Before */}
              <div className="space-y-2">
                <span className="text-amber text-xs uppercase tracking-wide font-semibold block">
                  Before starting
                </span>
                <ul className="space-y-1.5">
                  {phase.pre.map((item) => (
                    <li key={item} className="text-fg text-sm flex items-start gap-2">
                      <span className="text-amber shrink-0 mt-0.5">&rarr;</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* After */}
              <div className="space-y-2">
                <span className="text-green text-xs uppercase tracking-wide font-semibold block">
                  Done when
                </span>
                <ul className="space-y-1.5">
                  {phase.post.map((item) => (
                    <li key={item} className="text-fg text-sm flex items-start gap-2">
                      <span className="text-green shrink-0 mt-0.5">&check;</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
        </div>
      </div>
    </SlideShell>
  )
}
