import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

export function InsightSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell className="flex flex-col items-center justify-center text-center min-h-[80vh]">
      <div className="relative">
        <span className="text-6xl text-cyan opacity-20 absolute -top-10 -left-8 select-none font-serif">
          &ldquo;
        </span>

        <h2 className="text-3xl md:text-5xl font-bold text-fg leading-tight">
          {t.insight.quote}
        </h2>
        <h2 className="text-3xl md:text-5xl font-bold text-cyan glow-cyan leading-tight mt-2">
          {t.insight.quoteLine2}
        </h2>

        <span className="text-6xl text-cyan opacity-20 absolute -bottom-12 -right-8 select-none font-serif">
          &rdquo;
        </span>
      </div>

      <p className="text-lg text-muted max-w-2xl mx-auto mt-16 leading-relaxed">
        {t.insight.description}
      </p>
    </SlideShell>
  )
}
