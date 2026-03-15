import { useRef } from 'react';

export default function VideoPlayer({ src, poster, autoPlay, onEnded, className = '' }) {
  const videoRef = useRef(null);

  return (
    <div className={className} style={{ position: 'relative', width: '100%', height: '100%', background: '#000', borderRadius: 'var(--radius-md)', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <video
        ref={videoRef}
        src={src}
        poster={poster}
        autoPlay={autoPlay}
        controls
        playsInline
        onEnded={onEnded}
        style={{ width: '100%', maxHeight: '100%', objectFit: 'contain' }}
        controlsList="nodownload"
      />
    </div>
  );
}
