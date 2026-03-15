export default function VideoCard({ video, titleContext, className = '' }) {
  if (!video) return null;
  
  return (
    <div className={`card ${className}`} style={{ padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
      {titleContext && (
        <span className="badge badge-primary" style={{ alignSelf: 'flex-start' }}>
          {titleContext}
        </span>
      )}
      <h3 style={{ fontSize: 'var(--font-size-lg)', margin: 0 }} title={video.title}>
        {video.title?.length > 40 ? video.title.substring(0, 40) + '...' : video.title}
      </h3>
      
      <div style={{ display: 'flex', gap: 'var(--space-3)', flexWrap: 'wrap' }}>
        {video.duration_ms > 0 && (
          <span style={{ color: 'var(--color-text-muted)', fontSize: 'var(--font-size-sm)' }}>
            ⏱️ {(video.duration_ms / 1000).toFixed(1)}s
          </span>
        )}
        {video.status && (
          <span className={`badge ${video.status === 'active' ? 'badge-success' : 'badge-warning'}`}>
            {video.status}
          </span>
        )}
      </div>

      {video.schedules || video.views ? (
        <div style={{ fontSize: 'var(--font-size-sm)', color: 'var(--color-text-muted)' }}>
          {video.views !== undefined && <span>👁️ Views: {video.views} </span>}
        </div>
      ) : null}

      {video.url && (
        <div style={{ marginTop: 'auto', paddingTop: 'var(--space-2)' }}>
          <a 
            href={video.url} 
            target="_blank" 
            rel="noreferrer" 
            className="btn btn-ghost" 
            style={{ width: '100%', padding: 'var(--space-2)' }}
          >
            Preview Video ↗
          </a>
        </div>
      )}
    </div>
  );
}
