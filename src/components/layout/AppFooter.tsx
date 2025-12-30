import React from 'react';

// ============================================================================
// APP FOOTER COMPONENT
// Extracted from App.tsx for modular architecture
// ============================================================================

const styles = {
  footer: {
    textAlign: 'center' as const,
    padding: '32px 24px',
    borderTop: '1px solid #4A2C5A',
    color: '#7A7490',
    background: 'linear-gradient(180deg, transparent 0%, rgba(13, 8, 17, 0.5) 100%)',
  },
  footerLinks: {
    marginTop: '12px',
  },
  link: {
    color: '#7B5EA7',
    textDecoration: 'none',
    fontWeight: 500,
    transition: 'color 0.2s ease',
  },
};

export function AppFooter() {
  return (
    <footer style={styles.footer} data-testid="app-footer">
      <p>Chimera Pool - BlockDAG Mining Made Easy</p>
      <p style={styles.footerLinks}>
        <a
          href="https://awakening.bdagscan.com/"
          target="_blank"
          rel="noopener noreferrer"
          style={styles.link}
          data-testid="footer-explorer-link"
        >
          Block Explorer
        </a>
        {' | '}
        <a
          href="https://awakening.bdagscan.com/faucet"
          target="_blank"
          rel="noopener noreferrer"
          style={styles.link}
          data-testid="footer-faucet-link"
        >
          Faucet
        </a>
      </p>
    </footer>
  );
}

export default AppFooter;
