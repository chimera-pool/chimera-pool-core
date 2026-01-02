import React, { useState, useEffect } from 'react';
import { ComposableMap, Geographies, Geography, Marker, ZoomableGroup } from 'react-simple-maps';
import { colors, gradients } from '../../styles/shared';
import { formatHashrate } from '../../utils/formatters';

// ============================================================================
// GLOBAL MINER MAP COMPONENT
// Interactive world map displaying miner locations and statistics
// ============================================================================

const geoUrl = 'https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json';

export interface MinerLocation {
  city: string;
  country: string;
  countryCode: string;
  continent: string;
  lat: number;
  lng: number;
  minerCount: number;
  hashrate: number;
  activeCount: number;
  isActive: boolean;
}

export interface LocationStats {
  totalMiners: number;
  totalCountries: number;
  activeMiners: number;
  topCountries: { country: string; countryCode: string; minerCount: number; hashrate: number }[];
  continentBreakdown: { continent: string; minerCount: number; hashrate: number }[];
}

// Chimera theme continent colors
const CONTINENT_COLORS: { [key: string]: string } = {
  'North America': '#D4A84B',
  'South America': '#4ADE80',
  'Europe': '#7B5EA7',
  'Asia': '#FBBF24',
  'Africa': '#C45C5C',
  'Oceania': '#60A5FA',
  'Unknown': '#7A7490'
};

// Chimera Elite Theme Styles
const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
    borderRadius: '16px',
    padding: '24px',
    border: '1px solid #4A2C5A',
    marginBottom: '24px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    flexWrap: 'wrap' as const,
    gap: '16px',
  },
  title: {
    fontSize: '1.15rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  statsRow: {
    display: 'flex',
    gap: '12px',
  },
  statBadge: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    padding: '10px 18px',
    borderRadius: '10px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
  },
  statNumber: {
    color: '#D4A84B',
    fontSize: '1.3rem',
    fontWeight: 700,
  },
  statLabel: {
    color: '#B8B4C8',
    fontSize: '0.7rem',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    fontWeight: 500,
  },
  loading: {
    textAlign: 'center' as const,
    padding: '80px',
    color: '#D4A84B',
    fontSize: '0.95rem',
  },
  mapContainer: {
    display: 'flex',
    gap: '20px',
    flexWrap: 'wrap' as const,
  },
  mapWrapper: {
    flex: '1 1 600px',
    height: '400px',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
    borderRadius: '12px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    overflow: 'hidden',
    position: 'relative' as const,
    boxShadow: '0 4px 16px rgba(0, 0, 0, 0.3)',
  },
  tooltip: {
    position: 'fixed' as const,
    backgroundColor: colors.bgCard,
    border: `1px solid ${colors.primary}`,
    borderRadius: '8px',
    padding: '12px',
    zIndex: 1000,
    pointerEvents: 'none' as const,
  },
  tooltipCity: {
    color: colors.primary,
    fontWeight: 'bold',
    fontSize: '1rem',
  },
  tooltipCountry: {
    color: colors.textSecondary,
    fontSize: '0.85rem',
    marginBottom: '8px',
  },
  tooltipStats: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '4px',
    color: colors.textPrimary,
    fontSize: '0.85rem',
  },
  sidebar: {
    flex: '0 0 250px',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '20px',
  },
  sidebarSection: {
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    borderRadius: '14px',
    padding: '18px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    boxShadow: '0 4px 16px rgba(0, 0, 0, 0.2)',
  },
  sidebarTitle: {
    color: '#D4A84B',
    fontSize: '1rem',
    margin: '0 0 14px 0',
    fontWeight: 700,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    textShadow: '0 2px 4px rgba(0, 0, 0, 0.3)',
  },
  countryRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '8px 10px',
    borderBottom: `1px solid rgba(74, 44, 90, 0.3)`,
    borderRadius: '6px',
    transition: 'all 0.2s ease',
    cursor: 'default',
  },
  countryRowHover: {
    background: 'rgba(212, 168, 75, 0.1)',
    borderColor: 'rgba(212, 168, 75, 0.3)',
  },
  countryRank: {
    color: '#D4A84B',
    fontSize: '0.85rem',
    fontWeight: 700,
    width: '28px',
    textAlign: 'center' as const,
  },
  countryFlag: {
    fontSize: '1.1rem',
  },
  countryName: {
    flex: 1,
    color: colors.textPrimary,
    fontSize: '0.9rem',
    fontWeight: 500,
  },
  countryMiners: {
    color: colors.primary,
    fontWeight: 700,
    fontSize: '0.95rem',
  },
  continentRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    padding: '10px 12px',
    borderRadius: '8px',
    marginBottom: '6px',
    transition: 'all 0.2s ease',
    cursor: 'default',
  },
  continentRowHover: {
    background: 'rgba(212, 168, 75, 0.08)',
  },
  continentDot: {
    width: '12px',
    height: '12px',
    borderRadius: '50%',
    boxShadow: '0 0 8px currentColor',
  },
  continentName: {
    flex: 1,
    color: colors.textPrimary,
    fontSize: '0.9rem',
    fontWeight: 500,
  },
  continentMiners: {
    color: '#D4A84B',
    fontWeight: 600,
  },
  continentHashrate: {
    color: colors.textSecondary,
    fontSize: '0.75rem',
  },
};

export function GlobalMinerMap() {
  const [locations, setLocations] = useState<MinerLocation[]>([]);
  const [stats, setStats] = useState<LocationStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [hoveredLocation, setHoveredLocation] = useState<MinerLocation | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchData = async () => {
    try {
      const [locRes, statsRes] = await Promise.all([
        fetch('/api/v1/miners/locations'),
        fetch('/api/v1/miners/locations/stats')
      ]);

      if (locRes.ok) {
        const data = await locRes.json();
        setLocations(data.locations || []);
      }
      if (statsRes.ok) {
        const data = await statsRes.json();
        setStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch miner locations:', error);
    } finally {
      setLoading(false);
    }
  };

  const getMarkerSize = (count: number) => {
    if (count >= 10) return 12;
    if (count >= 5) return 9;
    if (count >= 2) return 7;
    return 5;
  };

  const handleMouseEnter = (location: MinerLocation, e: React.MouseEvent) => {
    setHoveredLocation(location);
    setTooltipPos({ x: e.clientX, y: e.clientY });
  };

  return (
    <section style={styles.section}>
      <div style={styles.header}>
        <h2 style={styles.title}>🌍 Global Miner Network</h2>
        <div style={styles.statsRow}>
          <div style={styles.statBadge}>
            <span style={styles.statNumber}>{stats?.totalMiners || 0}</span>
            <span style={styles.statLabel}>Total Miners</span>
          </div>
          <div style={styles.statBadge}>
            <span style={styles.statNumber}>{stats?.activeMiners || 0}</span>
            <span style={styles.statLabel}>Active</span>
          </div>
          <div style={styles.statBadge}>
            <span style={styles.statNumber}>{stats?.totalCountries || 0}</span>
            <span style={styles.statLabel}>Countries</span>
          </div>
        </div>
      </div>

      {loading ? (
        <div style={styles.loading}>Loading global miner network...</div>
      ) : (
        <div style={styles.mapContainer}>
          <div style={styles.mapWrapper}>
            <ComposableMap
              projection="geoMercator"
              projectionConfig={{ scale: 120, center: [0, 20] }}
              style={{ width: '100%', height: '100%', backgroundColor: colors.bgInput }}
            >
              <ZoomableGroup>
                <Geographies geography={geoUrl}>
                  {({ geographies }) =>
                    geographies.map((geo) => (
                      <Geography
                        key={geo.rsmKey}
                        geography={geo}
                        fill={colors.bgCard}
                        stroke={colors.border}
                        strokeWidth={0.5}
                        style={{
                          default: { outline: 'none' },
                          hover: { fill: colors.border, outline: 'none' },
                          pressed: { outline: 'none' }
                        }}
                      />
                    ))
                  }
                </Geographies>
                {/* Pool Server Markers */}
                <Marker coordinates={[-149.9003, 61.2181]}>
                  <g>
                    <polygon
                      points="0,-10 8,6 -8,6"
                      fill="#7B5EA7"
                      stroke="#F0EDF4"
                      strokeWidth={1.5}
                      style={{ cursor: 'pointer' }}
                    />
                    <title>Pool Server - Anchorage, AK</title>
                  </g>
                </Marker>
                {/* Miner Location Markers */}
                {locations.map((location, idx) => (
                  <Marker
                    key={idx}
                    coordinates={[location.lng, location.lat]}
                    onMouseEnter={(e) => handleMouseEnter(location, e as any)}
                    onMouseLeave={() => setHoveredLocation(null)}
                  >
                    <circle
                      r={getMarkerSize(location.minerCount)}
                      fill={location.isActive ? colors.primary : colors.textSecondary}
                      fillOpacity={0.8}
                      stroke={location.isActive ? colors.primary : colors.textSecondary}
                      strokeWidth={2}
                      strokeOpacity={0.4}
                      style={{ cursor: 'pointer' }}
                    >
                      {location.isActive && (
                        <animate
                          attributeName="r"
                          from={getMarkerSize(location.minerCount)}
                          to={getMarkerSize(location.minerCount) + 3}
                          dur="1.5s"
                          repeatCount="indefinite"
                        />
                      )}
                    </circle>
                  </Marker>
                ))}
              </ZoomableGroup>
            </ComposableMap>

            {hoveredLocation && (
              <div style={{
                ...styles.tooltip,
                left: tooltipPos.x + 10,
                top: tooltipPos.y - 60
              }}>
                <div style={styles.tooltipCity}>{hoveredLocation.city}</div>
                <div style={styles.tooltipCountry}>{hoveredLocation.country}</div>
                <div style={styles.tooltipStats}>
                  <span>⛏️ {hoveredLocation.minerCount} miners</span>
                  <span>⚡ {formatHashrate(hoveredLocation.hashrate)}</span>
                  <span>{hoveredLocation.activeCount} active</span>
                </div>
              </div>
            )}
          </div>

          <div style={styles.sidebar}>
            <div style={styles.sidebarSection} data-testid="top-countries-section">
              <h4 style={styles.sidebarTitle}>🏆 Top Countries</h4>
              {stats?.topCountries?.slice(0, 5).map((country, idx) => (
                <div 
                  key={idx} 
                  style={styles.countryRow}
                  data-testid={`country-row-${idx}`}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'rgba(212, 168, 75, 0.1)';
                    e.currentTarget.style.borderColor = 'rgba(212, 168, 75, 0.3)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'transparent';
                    e.currentTarget.style.borderColor = 'rgba(74, 44, 90, 0.3)';
                  }}
                >
                  <span style={styles.countryRank}>#{idx + 1}</span>
                  <span style={styles.countryName}>{country.country}</span>
                  <span style={styles.countryMiners}>{country.minerCount} ⛏️</span>
                </div>
              ))}
            </div>

            <div style={styles.sidebarSection} data-testid="continent-breakdown-section">
              <h4 style={styles.sidebarTitle}>🌐 By Continent</h4>
              {stats?.continentBreakdown?.map((cont, idx) => (
                <div 
                  key={idx} 
                  style={styles.continentRow}
                  data-testid={`continent-row-${idx}`}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'rgba(212, 168, 75, 0.08)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'transparent';
                  }}
                >
                  <span style={{ 
                    ...styles.continentDot, 
                    backgroundColor: CONTINENT_COLORS[cont.continent] || colors.textSecondary,
                    color: CONTINENT_COLORS[cont.continent] || colors.textSecondary
                  }}></span>
                  <span style={styles.continentName}>{cont.continent}</span>
                  <div style={{ textAlign: 'right' as const }}>
                    <span style={styles.continentMiners}>{cont.minerCount} ⛏️</span>
                    <div style={styles.continentHashrate}>{formatHashrate(cont.hashrate)}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

export default GlobalMinerMap;
