import React, { useState, useCallback } from 'react';
import { IGrafanaEmbedProps, buildGrafanaEmbedUrl } from '../interfaces/IGrafanaPanel';

/**
 * GrafanaEmbed - Embeds a Grafana panel via iframe
 * Provides loading state and error handling with callbacks
 */
export const GrafanaEmbed: React.FC<IGrafanaEmbedProps> = ({
  baseUrl,
  panel,
  className,
  style,
  onLoad,
  onError,
}) => {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  const embedUrl = buildGrafanaEmbedUrl(baseUrl, panel);

  const handleLoad = useCallback(() => {
    setIsLoading(false);
    setHasError(false);
    onLoad?.();
  }, [onLoad]);

  const handleError = useCallback(() => {
    setIsLoading(false);
    setHasError(true);
    onError?.(new Error(`Failed to load Grafana panel: ${panel.title}`));
  }, [onError, panel.title]);

  const containerStyle: React.CSSProperties = {
    position: 'relative',
    width: panel.width || '100%',
    height: panel.height || 280,
    backgroundColor: '#181B1F',
    borderRadius: '4px',
    overflow: 'hidden',
    ...style,
  };

  const iframeStyle: React.CSSProperties = {
    width: '100%',
    height: panel.height || 280,
    border: 'none',
    opacity: isLoading ? 0 : 1,
    transition: 'opacity 0.3s ease',
  };

  const loadingStyle: React.CSSProperties = {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: 'rgba(204, 204, 220, 0.65)',
    fontSize: '0.85rem',
    opacity: isLoading ? 1 : 0,
    transition: 'opacity 0.3s ease',
    pointerEvents: 'none',
  };

  const errorStyle: React.CSSProperties = {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#FF6B6B',
    fontSize: '0.85rem',
    padding: '20px',
    textAlign: 'center',
  };

  return (
    <div 
      data-testid="grafana-embed-container"
      className={className}
      style={containerStyle}
    >
      {isLoading && !hasError && (
        <div style={loadingStyle}>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
            <div 
              style={{ 
                width: '24px', 
                height: '24px', 
                border: '2px solid rgba(204, 204, 220, 0.2)', 
                borderTopColor: '#F5B800', 
                borderRadius: '50%', 
                animation: 'spin 1s linear infinite' 
              }} 
            />
            <span>Loading {panel.title}...</span>
          </div>
        </div>
      )}

      {hasError && (
        <div style={errorStyle}>
          <span style={{ marginBottom: '8px' }}>⚠️</span>
          <span>Failed to load chart</span>
          <span style={{ fontSize: '0.75rem', color: 'rgba(204, 204, 220, 0.5)', marginTop: '4px' }}>
            {panel.title}
          </span>
        </div>
      )}

      <iframe
        src={embedUrl}
        title={panel.title}
        style={iframeStyle}
        onLoad={handleLoad}
        onError={handleError}
        loading="lazy"
        sandbox="allow-scripts allow-same-origin"
      />
    </div>
  );
};

export default GrafanaEmbed;
