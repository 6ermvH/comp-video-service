import { useRef, useEffect, useImperativeHandle, forwardRef, useState } from 'react'

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
  const [leftEnded, setLeftEnded] = useState(false)
  const [rightEnded, setRightEnded] = useState(false)
  const endedRef = useRef({ left: false, right: false })

  const bothReady = leftReady && rightReady

  // Reset state when URLs change (new pair loaded)
  useEffect(() => {
    setLeftReady(false)
    setRightReady(false)
    setPlaying(false)
    setLeftEnded(false)
    setRightEnded(false)
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
    if (side === 'left') setLeftEnded(true)
    else setRightEnded(true)

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
    setLeftEnded(false)
    setRightEnded(false)

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
    aspectRatio: '16/9',
    background: '#000',
    borderRadius: '8px',
    display: 'block',
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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>

        {/* Left video */}
        <div style={{ position: 'relative' }}>
          <div style={{ position: 'absolute', top: '8px', left: '8px', zIndex: 10,
            background: 'rgba(0,0,0,0.7)', color: '#fff', fontSize: '13px',
            fontWeight: 700, padding: '3px 10px', borderRadius: '4px', letterSpacing: '0.05em' }}>
            A
          </div>
          {!leftReady && (
            <div style={loadingOverlayStyle}>
              <div className="spinner" style={{ width: 28, height: 28 }} />
            </div>
          )}
          <video
            ref={leftRef}
            src={leftUrl}
            style={videoStyle}
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
        <div style={{ position: 'relative' }}>
          <div style={{ position: 'absolute', top: '8px', left: '8px', zIndex: 10,
            background: 'rgba(0,0,0,0.7)', color: '#fff', fontSize: '13px',
            fontWeight: 700, padding: '3px 10px', borderRadius: '4px', letterSpacing: '0.05em' }}>
            B
          </div>
          {!rightReady && (
            <div style={loadingOverlayStyle}>
              <div className="spinner" style={{ width: 28, height: 28 }} />
            </div>
          )}
          <video
            ref={rightRef}
            src={rightUrl}
            style={videoStyle}
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

      {/* Controls */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px' }}>
        {!bothReady ? (
          <span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>
            Загрузка видео…
          </span>
        ) : (
          <>
            <button
              className="btn btn-ghost"
              onClick={togglePlayPause}
              style={{ minWidth: '100px' }}
            >
              {playing ? '⏸ Пауза' : '▶ Воспроизвести'}
            </button>
            <button
              className="btn btn-ghost"
              onClick={replay}
              title="Повторить (R)"
            >
              ↺ Повторить
            </button>
            {(leftEnded || rightEnded) && !playing && (
              <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
                Просмотр завершён
              </span>
            )}
          </>
        )}
      </div>
    </div>
  )
})

export default SyncVideoPlayer
