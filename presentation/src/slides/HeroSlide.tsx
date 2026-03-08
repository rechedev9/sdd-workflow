import type { ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'
import { Badge } from '../components/Badge'

export function HeroSlide(): ReactNode {
  const t = useTranslations()

  return (
    <SlideShell className="flex flex-col items-center justify-center text-center min-h-[80vh] gap-4">
      {/* Hook — the killer metric */}
      <p className="font-mono text-lg md:text-xl text-muted tracking-wide">
        {t.hero.hookLine}
      </p>
      <p className="text-sm text-cyan/70 font-mono mb-8">
        {t.hero.hookSub}
      </p>

      {/* Brand */}
      <h1 className="text-7xl md:text-9xl font-black text-fg glow-cyan leading-none tracking-tighter">
        {t.hero.title}
      </h1>

      <p className="text-lg md:text-xl text-muted mt-2 tracking-wide">
        {t.hero.subtitle}
      </p>

      <div className="mt-4">
        <Badge color="green">
          {t.hero.version} — {t.hero.versionLabel}
        </Badge>
      </div>

      <p className="text-base text-cyan font-mono mt-10 tracking-widest uppercase">
        {t.hero.tagline}
      </p>

      {/* Decorative gradient line */}
      <div className="w-full max-w-lg mt-8">
        <div
          className="h-px w-full"
          style={{
            background:
              'linear-gradient(90deg, transparent, var(--color-cyan), var(--color-green), var(--color-purple), transparent)',
          }}
        />
      </div>
    </SlideShell>
  )
}
