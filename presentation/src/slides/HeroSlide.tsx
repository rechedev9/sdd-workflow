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

      {/* Quick links */}
      <div className="flex items-center gap-6 mt-4">
        <a
          href="https://github.com/rechedev9/sdd-workflow"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-2 text-muted hover:text-fg transition-colors text-sm font-mono"
        >
          <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0016 8c0-4.42-3.58-8-8-8z"/>
          </svg>
          github.com/rechedev9/sdd-workflow
        </a>

        <span className="text-muted/30">|</span>

        <a
          href="https://discord.com/users/raisen1340"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-2 text-muted hover:text-[#5865F2] transition-colors text-sm font-mono"
        >
          <svg width="18" height="14" viewBox="0 0 71 55" fill="none" aria-hidden="true">
            <path d="M60.1 4.9A58.5 58.5 0 0045.4.2a.2.2 0 00-.2.1 40.8 40.8 0 00-1.8 3.7 54 54 0 00-16.2 0A39 39 0 0025.4.3a.2.2 0 00-.2-.1A58.4 58.4 0 0010.5 4.9a.2.2 0 00-.1.1C1.5 18.7-.9 32.2.3 45.5v.1a58.8 58.8 0 0017.7 9a.2.2 0 00.3-.1 42.1 42.1 0 003.6-5.9.2.2 0 00-.1-.3 38.8 38.8 0 01-5.5-2.6.2.2 0 01 0-.4l1.1-.9a.2.2 0 01.2 0 42 42 0 0035.6 0 .2.2 0 01.2 0l1.1.9a.2.2 0 010 .4c-1.8 1-3.6 1.9-5.5 2.6a.2.2 0 00-.1.3 47.3 47.3 0 003.6 5.9.2.2 0 00.3.1A58.6 58.6 0 0070.5 45.6v-.1c1.4-15-2.3-28-9.8-39.6a.2.2 0 00-.1 0zM23.7 37.3c-3.4 0-6.3-3.2-6.3-7s2.8-7 6.3-7 6.4 3.1 6.3 7-2.8 7-6.3 7zm23.2 0c-3.4 0-6.3-3.2-6.3-7s2.8-7 6.3-7 6.4 3.1 6.3 7-2.8 7-6.3 7z" fill="currentColor"/>
          </svg>
          raisen1340
        </a>
      </div>
    </SlideShell>
  )
}
