import { createContext, useContext, useState, type ReactNode } from 'react'
import { en, type Translations } from './en'
import { es } from './es'

export type Language = 'en' | 'es'
export type { Translations }

const translations: Record<Language, Translations> = { en, es }

type LanguageContextValue = {
  readonly language: Language
  readonly setLanguage: (l: Language) => void
  readonly t: Translations
}

const LanguageContext = createContext<LanguageContextValue>({
  language: 'en',
  setLanguage: () => {},
  t: en,
})

export function LanguageProvider({ children }: { children: ReactNode }): ReactNode {
  const [language, setLanguage] = useState<Language>('en')
  const t = translations[language]
  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useTranslations(): Translations {
  return useContext(LanguageContext).t
}

export function useLanguage(): { readonly language: Language; readonly setLanguage: (l: Language) => void } {
  const { language, setLanguage } = useContext(LanguageContext)
  return { language, setLanguage }
}
