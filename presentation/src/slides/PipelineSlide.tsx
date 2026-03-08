import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Badge } from '../components/Badge'

type PhaseData = {
  readonly name: string
  readonly description: string
  readonly v11: boolean
}

function PhaseNode({ phase }: { readonly phase: PhaseData }): ReactNode {
  return (
    <div className="group relative shrink-0">
      <div
        className={`flex items-center gap-2 bg-surface border ${
          phase.v11 ? 'border-cyan/50' : 'border-edge'
        } rounded-lg px-4 py-2.5 cursor-default`}
      >
        <span className={`font-mono text-sm font-semibold ${phase.v11 ? 'text-cyan' : 'text-fg'}`}>
          {phase.name}
        </span>
        {phase.v11 ? <Badge color="green">v1.1</Badge> : null}
      </div>

      {/* Tooltip */}
      <div className="
        absolute bottom-full left-1/2 -translate-x-1/2 mb-3 z-50
        w-52 bg-panel border border-edge rounded-lg p-3 shadow-xl shadow-black/40
        text-xs text-muted text-center leading-relaxed
        pointer-events-none
        opacity-0 translate-y-1
        group-hover:opacity-100 group-hover:translate-y-0
        transition-all duration-200 ease-out
      ">
        {phase.description}
      </div>
    </div>
  )
}

function Arrow(): ReactNode {
  return <span className="text-edge font-mono text-base shrink-0">→</span>
}

function DownArrow(): ReactNode {
  return (
    <div className="flex justify-center py-2">
      <span className="text-edge font-mono text-base">↓</span>
    </div>
  )
}

function PhaseRow({ phases }: { readonly phases: readonly PhaseData[] }): ReactNode {
  return (
    <div className="flex items-center justify-center gap-2">
      {phases.map((phase, i) => (
        <div key={phase.name} className="flex items-center gap-2">
          {i > 0 ? <Arrow /> : null}
          <PhaseNode phase={phase} />
        </div>
      ))}
    </div>
  )
}

export function PipelineSlide(): ReactNode {
  const t = useTranslations()
  const phases = t.pipeline.phases

  // row 1: init → explore → propose
  const row1 = phases.slice(0, 3)
  // row 2: spec ⇄ design (parallel)
  const parallel = phases.slice(3, 5)
  // row 3: tasks → apply → review → verify
  const row3 = phases.slice(5, 9)
  // row 4: clean → archive
  const row4 = phases.slice(9, 11)

  return (
    <SlideShell>
      {/* Header */}
      <div className="mb-8 text-center">
        <h2 className="text-4xl font-bold text-fg mb-2">{t.pipeline.title}</h2>
        <p className="text-muted text-lg">{t.pipeline.subtitle}</p>
      </div>

      {/* Legend */}
      <div className="flex justify-center mb-6">
        <span className="flex items-center gap-1.5 text-xs text-muted">
          <span className="w-2.5 h-2.5 rounded border border-cyan bg-cyan/10 inline-block" />
          {t.pipeline.v11Badge}
        </span>
      </div>

      {/* Pipeline flow */}
      <div className="space-y-0">
        <PhaseRow phases={row1} />
        <DownArrow />

        {/* Parallel block */}
        <div className="flex items-center justify-center gap-3">
          {parallel.map((phase, i) => (
            <div key={phase.name} className="flex items-center gap-3">
              {i > 0 ? (
                <div className="flex flex-col items-center gap-0.5">
                  <span className="text-cyan font-mono text-xs leading-none">{t.pipeline.parallel}</span>
                  <span className="text-cyan font-mono text-sm">⇄</span>
                </div>
              ) : null}
              <PhaseNode phase={phase} />
            </div>
          ))}
        </div>

        <DownArrow />
        <PhaseRow phases={row3} />
        <DownArrow />
        <PhaseRow phases={row4} />
      </div>
    </SlideShell>
  )
}
