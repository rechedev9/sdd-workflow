import { useState, useEffect, useCallback } from 'react'

type SlideNavigation = {
  readonly current: number
  readonly next: () => void
  readonly prev: () => void
  readonly goTo: (index: number) => void
  readonly total: number
}

const STORAGE_KEY = 'deck-slide'

export function useSlideNavigation(totalSlides: number): SlideNavigation {
  const [current, setCurrent] = useState<number>(() => {
    const saved = sessionStorage.getItem(STORAGE_KEY)
    const n = saved !== null ? parseInt(saved, 10) : 0
    return Number.isNaN(n) ? 0 : Math.min(n, totalSlides - 1)
  })

  useEffect(() => {
    sessionStorage.setItem(STORAGE_KEY, String(current))
  }, [current])

  const next = useCallback((): void => {
    setCurrent(c => Math.min(c + 1, totalSlides - 1))
  }, [totalSlides])

  const prev = useCallback((): void => {
    setCurrent(c => Math.max(c - 1, 0))
  }, [])

  const goTo = useCallback((index: number): void => {
    setCurrent(Math.max(0, Math.min(index, totalSlides - 1)))
  }, [totalSlides])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent): void => {
      if (e.key === 'ArrowRight' || e.key === ' ') {
        e.preventDefault()
        next()
      }
      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        prev()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [next, prev])

  return { current, next, prev, goTo, total: totalSlides }
}
