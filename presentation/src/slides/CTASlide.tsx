import { useState, useCallback, type ReactNode } from 'react'
import { useTranslations } from '../i18n/context'
import { SlideShell } from '../components/SlideShell'

export function CTASlide(): ReactNode {
  const t = useTranslations()
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback((): void => {
    void navigator.clipboard.writeText(t.cta.discordTag)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [t.cta.discordTag])

  return (
    <SlideShell>
      <div className="max-w-2xl mx-auto space-y-8">
        {/* Motivation post */}
        <div className="space-y-5">
          <h2 className="text-3xl font-black text-fg">{t.cta.title}</h2>
          <p className="text-muted text-base leading-relaxed italic">
            &ldquo;{t.cta.motivation}&rdquo;
          </p>
          <p className="text-cyan text-lg font-semibold">{t.cta.closing}</p>
        </div>

        {/* Divider */}
        <div className="w-32 h-px bg-gradient-to-r from-cyan via-purple to-green opacity-50" />

        {/* Author + Discord */}
        <div className="flex items-center justify-between flex-wrap gap-6">
          {/* Attribution */}
          <p className="text-muted text-sm">
            {t.cta.designedBy}{' '}
            <span className="text-fg font-bold text-base">{t.cta.author}</span>
          </p>

          {/* Discord */}
          <button
            type="button"
            onClick={handleCopy}
            className="flex items-center gap-3 bg-[#5865F2] hover:bg-[#4752C4] text-white font-semibold rounded-lg px-5 py-2.5 text-sm transition-colors cursor-pointer"
          >
            <svg width="20" height="15" viewBox="0 0 71 55" fill="none" aria-hidden="true">
              <path d="M60.1 4.9A58.5 58.5 0 0045.4.2a.2.2 0 00-.2.1 40.8 40.8 0 00-1.8 3.7 54 54 0 00-16.2 0A39 39 0 0025.4.3a.2.2 0 00-.2-.1A58.4 58.4 0 0010.5 4.9a.2.2 0 00-.1.1C1.5 18.7-.9 32.2.3 45.5v.1a58.8 58.8 0 0017.7 9a.2.2 0 00.3-.1 42.1 42.1 0 003.6-5.9.2.2 0 00-.1-.3 38.8 38.8 0 01-5.5-2.6.2.2 0 01 0-.4l1.1-.9a.2.2 0 01.2 0 42 42 0 0035.6 0 .2.2 0 01.2 0l1.1.9a.2.2 0 010 .4c-1.8 1-3.6 1.9-5.5 2.6a.2.2 0 00-.1.3 47.3 47.3 0 003.6 5.9.2.2 0 00.3.1A58.6 58.6 0 0070.5 45.6v-.1c1.4-15-2.3-28-9.8-39.6a.2.2 0 00-.1 0zM23.7 37.3c-3.4 0-6.3-3.2-6.3-7s2.8-7 6.3-7 6.4 3.1 6.3 7-2.8 7-6.3 7zm23.2 0c-3.4 0-6.3-3.2-6.3-7s2.8-7 6.3-7 6.4 3.1 6.3 7-2.8 7-6.3 7z" fill="currentColor"/>
            </svg>
            <span className="font-mono">
              {copied ? t.cta.copied : t.cta.discordTag}
            </span>
          </button>
        </div>
      </div>
    </SlideShell>
  )
}
