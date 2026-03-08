import { useState, useEffect, useCallback, type ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

const STAT_ACCENTS = [
  'text-cyan',
  'text-green',
  'text-purple',
  'text-cyan',
  'text-green',
  'text-purple',
  'text-amber',
  'text-cyan',
  'text-green',
] as const

function LaunchOverlay({ onDone }: { readonly onDone: () => void }): ReactNode {
  useEffect(() => {
    const timer = setTimeout(() => {
      window.open('https://gravityroom.app', '_blank', 'noopener,noreferrer')
      onDone()
    }, 1700)
    return () => clearTimeout(timer)
  }, [onDone])

  return (
    <div className="anime-overlay fixed inset-0 z-50 flex flex-col items-center justify-center overflow-hidden">
      {/* Speed lines */}
      <div className="anime-speed-lines absolute inset-0 pointer-events-none" />

      {/* Shockwave rings */}
      <div className="absolute inset-0 pointer-events-none">
        <span className="anime-ring anime-ring-1" />
        <span className="anime-ring anime-ring-2" />
        <span className="anime-ring anime-ring-3" />
      </div>

      {/* "— Launching —" label */}
      <p className="anime-launching-label relative z-10 text-xs font-mono uppercase text-cyan/60 mb-5 tracking-[0.4em]">
        — Launching —
      </p>

      {/* URL slams in */}
      <div className="anime-url-slam relative z-10 text-center px-4">
        <span className="anime-url-text text-5xl md:text-7xl font-black font-mono text-cyan">
          gravityroom.app
        </span>
      </div>

      {/* Energy charge bar */}
      <div className="anime-energy-bar relative z-10 mt-8 w-56 h-px bg-edge rounded-full overflow-hidden">
        <div className="anime-energy-fill rounded-full bg-cyan" />
      </div>
    </div>
  )
}

export function CaseStudySlide(): ReactNode {
  const t = useTranslations()
  const [launching, setLaunching] = useState(false)

  const handleLaunch = useCallback((): void => {
    setLaunching(true)
  }, [])

  const handleDone = useCallback((): void => {
    setLaunching(false)
  }, [])

  return (
    <>
      {launching ? <LaunchOverlay onDone={handleDone} /> : null}

      <SlideShell>
        <div className="space-y-6">
          {/* Header */}
          <div className="text-center space-y-2">
            <p className="text-xs uppercase tracking-widest text-cyan font-mono">
              {t.caseStudy.title}
            </p>
            <button
              type="button"
              onClick={handleLaunch}
              className="group cursor-pointer bg-transparent border-0 p-0"
            >
              <h2 className="text-4xl md:text-6xl font-black text-fg group-hover:text-purple transition-colors duration-300 glow-purple">
                {t.caseStudy.projectName}
                <span className="text-2xl md:text-3xl text-purple/60 group-hover:text-purple ml-3 transition-colors duration-300">
                  ↗
                </span>
              </h2>
            </button>
            <p className="text-base text-muted max-w-2xl mx-auto">
              {t.caseStudy.projectDesc}
            </p>
          </div>

          {/* Metrics Grid — 3×3 */}
          <div className="grid grid-cols-3 gap-3">
            {t.caseStudy.stats.map((stat, i) => {
              const accent = STAT_ACCENTS[i % STAT_ACCENTS.length]
              return (
                <div
                  key={stat.label}
                  className="bg-surface border border-edge rounded-lg p-3 text-center card-glow"
                >
                  <div className={`text-xl md:text-2xl font-black font-mono leading-none ${accent}`}>
                    {stat.value}
                  </div>
                  <div className="text-xs text-muted mt-1 leading-tight">
                    {stat.label}
                  </div>
                </div>
              )
            })}
          </div>

          {/* Tech Stack */}
          <div className="flex flex-wrap justify-center gap-2">
            {t.caseStudy.techStack.map((tech) => (
              <span
                key={tech}
                className="bg-panel border border-edge rounded-full px-3 py-1 text-xs font-mono text-fg/80"
              >
                {tech}
              </span>
            ))}
          </div>

          {/* SDD Note */}
          <p className="text-center text-sm text-green font-mono">
            {t.caseStudy.sddNote}
          </p>
        </div>
      </SlideShell>
    </>
  )
}
