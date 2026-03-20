import { useRef, useEffect, useImperativeHandle, forwardRef, useState } from 'react'
import { useWindowWidth } from '../hooks/useWindowWidth.js'

/**
 * Synchronized dual-video player.
 * Both videos start/pause together. Replay resets both to start.
 *
 * Props:
 *   leftUrl    — presigned URL for left video
 *   rightUrl   — presigned URL for right video
 *   onReplay   — called whenever replay happens
 *   onEnded    — called when both videos have ended
 */
const SyncVideoPlayer = forwardRef(function SyncVideoPlayer(
  { leftUrl, rightUrl, onReplay, onEnded },
  ref
) {
  const leftRef = useRef(null)
  const rightRef = useRef(null)
  const [leftReady, setLeftReady] = useState(false)
  const [rightReady, setRightReady] = useState(false)
  const [playing, setPlaying] = useState(false)
  const endedRef = useRef({ left: false, right: false })

  const isMobile = useWindowWidth() <= 768
  const bothReady = leftReady && rightReady

  // Reset state when URLs change (new pair loaded)
  useEffect(() => {
    setLeftReady(false)
    setRightReady(false)
    setPlaying(false)
    endedRef.current = { left: false, right: false }
  }, [leftUrl, rightUrl])

  // Auto-play when both are ready
  useEffect(() => {
    if (!bothReady) return
    const left = leftRef.current
    const right = rightRef.current
    if (!left || !right) return

    left.currentTime = 0
    right.currentTime = 0

    Promise.all([left.play(), right.play()])
      .then(() => setPlaying(true))
      .catch(() => { /* autoplay blocked — user must click */ })
  }, [bothReady])

  const handleEnded = (side) => {
    endedRef.current[side] = true

    if (endedRef.current.left && endedRef.current.right) {
      setPlaying(false)
      onEnded?.()
    }
  }

  const replay = () => {
    const left = leftRef.current
    const right = rightRef.current
    if (!left || !right) return

    left.currentTime = 0
    right.currentTime = 0
    endedRef.current = { left: false, right: false }

    Promise.all([left.play(), right.play()])
      .then(() => setPlaying(true))
      .catch(() => {})

    onReplay?.()
  }

  const togglePlayPause = () => {
    const left = leftRef.current
    const right = rightRef.current
    if (!left || !right) return

    if (playing) {
      left.pause()
      right.pause()
      setPlaying(false)
    } else {
      Promise.all([left.play(), right.play()])
        .then(() => setPlaying(true))
        .catch(() => {})
    }
  }

  // Expose replay() to parent via ref
  useImperativeHandle(ref, () => ({ replay, togglePlayPause }))

  const videoStyle = {
    width: '100%',
    height: '100%',
    background: '#000',
    borderRadius: '8px',
    display: 'block',
    objectFit: 'contain',
  }

  const loadingOverlayStyle = {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'rgba(0,0,0,0.6)',
    borderRadius: '8px',
  }

  const videosContainerStyle = isMobile
    ? { display: 'flex', flexDirection: 'column', gap: '8px', width: '100%' }
    : {
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '0px',
        flex: '1 1 auto',
        minHeight: 0,
        height: 'clamp(510px, 71vh, 930px)',
      }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', width: '100%', height: '100%' }}>
      <div style={videosContainerStyle}>

        {/* Left video */}
        <div style={{ position: 'relative', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text)',
            fontSize: '15px',
            fontWeight: 700,
            marginBottom: '4px',
            display: isMobile ? 'block' : 'none',
          }}>
            Вариант A
          </div>
          {!leftReady && (
            <div style={loadingOverlayStyle}>
              <div className="spinner" style={{ width: 28, height: 28 }} />
            </div>
          )}
          <video
            ref={leftRef}
            src={leftUrl}
            style={{ ...videoStyle, objectPosition: isMobile ? 'center' : 'right center' }}
            preload="auto"
            playsInline
            controlsList="nodownload nofullscreen"
            disablePictureInPicture
            onCanPlayThrough={() => setLeftReady(true)}
            onEnded={() => handleEnded('left')}
            onClick={togglePlayPause}
          />
        </div>

        {/* Right video */}
        <div style={{ position: 'relative', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text)',
            fontSize: '15px',
            fontWeight: 700,
            marginBottom: '4px',
            display: isMobile ? 'block' : 'none',
          }}>
            Вариант B
          </div>
          {!rightReady && (
            <div style={loadingOverlayStyle}>
              <div className="spinner" style={{ width: 28, height: 28 }} />
            </div>
          )}
          <video
            ref={rightRef}
            src={rightUrl}
            style={{ ...videoStyle, objectPosition: isMobile ? 'center' : 'left center' }}
            preload="auto"
            playsInline
            controlsList="nodownload nofullscreen"
            disablePictureInPicture
            onCanPlayThrough={() => setRightReady(true)}
            onEnded={() => handleEnded('right')}
            onClick={togglePlayPause}
          />
        </div>
      </div>

      {/* Labels + controls */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: isMobile ? '1fr' : '1fr auto 1fr',
        alignItems: 'center',
        gap: '8px',
      }}>
        {!isMobile && (
          <div style={{ textAlign: 'center', color: 'var(--color-text)', fontSize: '15px', fontWeight: 700 }}>
            Вариант A
          </div>
        )}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px' }}>
          <button
            className="btn btn-ghost"
            onClick={togglePlayPause}
            disabled={!bothReady}
            style={{ width: isMobile ? '100%' : '138px', height: '34px', padding: '6px 10px', justifyContent: 'center', fontSize: '13px', lineHeight: 1 }}
          >
            {playing ? '⏸ Пауза' : '▶ Воспроизвести'}
          </button>
          {!isMobile && (
            <button
              className="btn btn-ghost"
              onClick={replay}
              disabled={!bothReady}
              title="Повторить (R)"
              style={{ width: '138px', minHeight: '34px', padding: '6px 10px', justifyContent: 'center', fontSize: '13px' }}
            >
              ↺ Повторить
            </button>
          )}
        </div>
        {!isMobile && (
          <div style={{ textAlign: 'center', color: 'var(--color-text)', fontSize: '15px', fontWeight: 700 }}>
            Вариант B
          </div>
        )}
      </div>
    </div>
  )
})

export default SyncVideoPlayer
