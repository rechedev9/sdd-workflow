import type { ReactNode } from 'react'
import { useLanguage, type Language } from '../i18n/context'

const languages: readonly Language[] = ['en', 'es']

export function LanguageToggle(): ReactNode {
  const { language, setLanguage } = useLanguage()

  return (
    <div className="fixed top-4 right-4 z-50 flex items-center gap-1">
      {languages.map((lang) => (
        <button
          key={lang}
          type="button"
          onClick={() => setLanguage(lang)}
          className={`px-3 py-1.5 text-sm rounded-full uppercase transition-colors ${
            lang === language
              ? 'bg-cyan text-base font-bold'
              : 'bg-surface text-muted border border-edge hover:border-muted'
          }`}
        >
          {lang}
        </button>
      ))}
    </div>
  )
}
