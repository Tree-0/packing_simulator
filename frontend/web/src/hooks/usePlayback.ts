import { useCallback, useEffect, useState } from 'react'

export const PLAYBACK_SPEEDS = [0.25, 0.5, 1, 2, 4] as const

export interface PlaybackController {
  frameIndex: number
  isPlaying: boolean
  speed: number
  play: () => void
  pause: () => void
  restart: () => void
  previous: () => void
  next: () => void
  seek: (frameIndex: number) => void
  setSpeed: (speed: number) => void
}

export function usePlayback(frameCount: number, frameDelayMs: number): PlaybackController {
  const lastFrame = Math.max(0, frameCount - 1)
  const [frameIndex, setFrameIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(frameCount > 1)
  const [speed, setSpeedState] = useState(1)

  useEffect(() => {
    setFrameIndex((current) => Math.min(current, lastFrame))
    if (frameCount <= 1) {
      setIsPlaying(false)
    }
  }, [frameCount, lastFrame])

  useEffect(() => {
    if (!isPlaying || frameIndex >= lastFrame) {
      if (isPlaying && frameIndex >= lastFrame) {
        setIsPlaying(false)
      }
      return undefined
    }

    const timer = window.setTimeout(() => {
      setFrameIndex((current) => Math.min(current + 1, lastFrame))
    }, frameDelayMs / speed)
    return () => window.clearTimeout(timer)
  }, [frameDelayMs, frameIndex, isPlaying, lastFrame, speed])

  const play = useCallback(() => {
    setFrameIndex((current) => (current >= lastFrame ? 0 : current))
    if (lastFrame > 0) {
      setIsPlaying(true)
    }
  }, [lastFrame])

  const pause = useCallback(() => setIsPlaying(false), [])
  const restart = useCallback(() => {
    setFrameIndex(0)
    setIsPlaying(lastFrame > 0)
  }, [lastFrame])
  const previous = useCallback(() => {
    setIsPlaying(false)
    setFrameIndex((current) => Math.max(0, current - 1))
  }, [])
  const next = useCallback(() => {
    setIsPlaying(false)
    setFrameIndex((current) => Math.min(lastFrame, current + 1))
  }, [lastFrame])
  const seek = useCallback(
    (nextFrame: number) => {
      setIsPlaying(false)
      setFrameIndex(Math.max(0, Math.min(lastFrame, nextFrame)))
    },
    [lastFrame],
  )
  const setSpeed = useCallback((nextSpeed: number) => {
    if (PLAYBACK_SPEEDS.some((candidate) => candidate === nextSpeed)) {
      setSpeedState(nextSpeed)
    }
  }, [])

  return { frameIndex, isPlaying, speed, play, pause, restart, previous, next, seek, setSpeed }
}
