import type { ReactNode } from 'react'
import { LanguageProvider } from './i18n/context'
import { useSlideNavigation } from './hooks/useSlideNavigation'
import { Navigation } from './components/Navigation'
import { LanguageToggle } from './components/LanguageToggle'
import { slides } from './slides'

function Deck(): ReactNode {
  const { current, next, prev, total } = useSlideNavigation(slides.length)
  const CurrentSlide = slides[current]

  if (!CurrentSlide) {
    return null
  }

  return (
    <>
      <LanguageToggle />
      <div key={current}>
        <CurrentSlide />
      </div>
      <Navigation current={current} total={total} onPrev={prev} onNext={next} />
    </>
  )
}

export function App(): ReactNode {
  return (
    <LanguageProvider>
      <Deck />
    </LanguageProvider>
  )
}
