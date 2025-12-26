import React from 'react';

// ============================================================================
// CHIMERA POOL LOGO COMPONENT
// SVG representation of the mythological Chimera (Lion, Goat, Serpent)
// ============================================================================

interface ChimeraLogoProps {
  size?: number;
  className?: string;
  style?: React.CSSProperties;
}

export const ChimeraLogo: React.FC<ChimeraLogoProps> = ({ 
  size = 48, 
  className,
  style 
}) => {
  return (
    <svg 
      width={size} 
      height={size} 
      viewBox="0 0 100 100" 
      fill="none" 
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      style={style}
    >
      <defs>
        {/* Gradients for the three creatures */}
        <linearGradient id="lionGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#E8C171" />
          <stop offset="50%" stopColor="#D4A84B" />
          <stop offset="100%" stopColor="#B8923A" />
        </linearGradient>
        <linearGradient id="goatGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#9B7EC7" />
          <stop offset="50%" stopColor="#7B5EA7" />
          <stop offset="100%" stopColor="#5A4580" />
        </linearGradient>
        <linearGradient id="serpentGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#E07777" />
          <stop offset="50%" stopColor="#C45C5C" />
          <stop offset="100%" stopColor="#A04545" />
        </linearGradient>
        <linearGradient id="bodyGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#4A2C5A" />
          <stop offset="100%" stopColor="#2D1F3D" />
        </linearGradient>
        {/* Glow effect */}
        <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>
      
      {/* Background circle */}
      <circle cx="50" cy="50" r="46" fill="url(#bodyGradient)" stroke="#4A2C5A" strokeWidth="2"/>
      
      {/* Serpent body - curved S shape on the right */}
      <path 
        d="M 65 70 Q 80 55 70 40 Q 60 25 75 15" 
        stroke="url(#serpentGradient)" 
        strokeWidth="6" 
        strokeLinecap="round"
        fill="none"
        filter="url(#glow)"
      />
      <circle cx="75" cy="15" r="4" fill="url(#serpentGradient)" /> {/* Serpent head */}
      <circle cx="77" cy="13" r="1.5" fill="#FF6B6B" /> {/* Serpent eye */}
      
      {/* Goat head - center top */}
      <ellipse cx="50" cy="35" rx="12" ry="14" fill="url(#goatGradient)" filter="url(#glow)"/>
      {/* Goat horns */}
      <path d="M 42 28 Q 35 15 38 8" stroke="url(#goatGradient)" strokeWidth="3" strokeLinecap="round" fill="none"/>
      <path d="M 58 28 Q 65 15 62 8" stroke="url(#goatGradient)" strokeWidth="3" strokeLinecap="round" fill="none"/>
      {/* Goat eyes */}
      <circle cx="46" cy="33" r="2" fill="#F0EDF4"/>
      <circle cx="54" cy="33" r="2" fill="#F0EDF4"/>
      <circle cx="46" cy="33" r="1" fill="#2D1F3D"/>
      <circle cx="54" cy="33" r="1" fill="#2D1F3D"/>
      {/* Goat beard */}
      <path d="M 50 45 L 50 52" stroke="url(#goatGradient)" strokeWidth="2" strokeLinecap="round"/>
      
      {/* Lion head - left side */}
      <ellipse cx="30" cy="50" rx="14" ry="12" fill="url(#lionGradient)" filter="url(#glow)"/>
      {/* Lion mane */}
      <path 
        d="M 18 38 Q 12 45 15 55 Q 12 60 18 65 Q 25 72 35 68 Q 42 65 44 58" 
        stroke="url(#lionGradient)" 
        strokeWidth="4" 
        strokeLinecap="round"
        fill="none"
        opacity="0.8"
      />
      {/* Lion features */}
      <circle cx="26" cy="48" r="2" fill="#2D1F3D"/> {/* Eye */}
      <ellipse cx="22" cy="54" rx="3" ry="2" fill="#B8923A"/> {/* Nose */}
      {/* Lion mouth/roar */}
      <path d="M 18 56 Q 22 60 18 62" stroke="#A04545" strokeWidth="1.5" strokeLinecap="round" fill="none"/>
      
      {/* Central swirl connecting all three */}
      <path 
        d="M 40 55 Q 50 65 60 55 Q 70 45 65 70" 
        stroke="url(#bodyGradient)" 
        strokeWidth="8" 
        strokeLinecap="round"
        fill="none"
        opacity="0.6"
      />
    </svg>
  );
};

// Text logo with the Chimera icon
export const ChimeraLogoFull: React.FC<{ iconSize?: number; fontSize?: string }> = ({ 
  iconSize = 40,
  fontSize = '1.75rem'
}) => {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
      <ChimeraLogo size={iconSize} />
      <div>
        <h1 style={{ 
          fontSize, 
          margin: 0, 
          color: '#D4A84B', 
          fontWeight: 700, 
          letterSpacing: '-0.02em',
          fontFamily: "'Inter', sans-serif"
        }}>
          Chimera Pool
        </h1>
        <p style={{ 
          fontSize: '0.85rem', 
          color: '#B8B4C8', 
          margin: '2px 0 0', 
          letterSpacing: '0.03em',
          fontWeight: 500
        }}>
          Elite Mining Platform
        </p>
      </div>
    </div>
  );
};

export default ChimeraLogo;
