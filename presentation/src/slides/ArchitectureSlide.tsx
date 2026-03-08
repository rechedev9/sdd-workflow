import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

type ArchKey = 'subAgents' | 'artifacts' | 'memory' | 'skills'

const COMPONENTS: readonly { readonly key: ArchKey; readonly color: string; readonly borderColor: string }[] = [
  { key: 'subAgents', color: 'text-green', borderColor: 'border-t-green' },
  { key: 'artifacts', color: 'text-purple', borderColor: 'border-t-purple' },
  { key: 'memory', color: 'text-amber', borderColor: 'border-t-amber' },
  { key: 'skills', color: 'text-cyan', borderColor: 'border-t-cyan' },
]

export function ArchitectureSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell>
      <div className="mb-12 text-center">
        <h2 className="text-4xl font-bold text-fg">{t.architecture.title}</h2>
      </div>

      {/* Orchestrator - central node */}
      <div className="flex justify-center mb-2">
        <div className="bg-surface border-2 border-cyan rounded-lg p-6 max-w-md w-full text-center card-glow">
          <h3 className="text-xl font-semibold text-cyan glow-cyan">
            {t.architecture.orchestrator.title}
          </h3>
          <p className="text-muted text-sm mt-2 leading-relaxed">
            {t.architecture.orchestrator.description}
          </p>
        </div>
      </div>

      {/* Connector arrows */}
      <div className="flex justify-center my-1">
        <div className="flex items-center gap-8">
          <span className="text-edge text-2xl">|</span>
          <span className="text-edge text-2xl">|</span>
          <span className="text-edge text-2xl">|</span>
          <span className="text-edge text-2xl">|</span>
        </div>
      </div>
      <div className="flex justify-center mb-2">
        <div className="flex items-center gap-8">
          <span className="text-green text-lg font-mono">&#x2193;</span>
          <span className="text-purple text-lg font-mono">&#x2193;</span>
          <span className="text-amber text-lg font-mono">&#x2193;</span>
          <span className="text-cyan text-lg font-mono">&#x2193;</span>
        </div>
      </div>

      {/* Four surrounding components */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-3xl mx-auto">
        {COMPONENTS.map(({ key, color, borderColor }) => (
          <div
            key={key}
            className={`bg-surface border border-edge border-t-2 ${borderColor} rounded-lg p-4 card-glow`}
          >
            <h4 className={`font-semibold ${color} mb-1`}>
              {t.architecture[key].title}
            </h4>
            <p className="text-muted text-sm leading-relaxed">
              {t.architecture[key].description}
            </p>
          </div>
        ))}
      </div>
    </SlideShell>
  )
}
