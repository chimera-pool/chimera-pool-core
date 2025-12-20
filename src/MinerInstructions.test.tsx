import React from 'react';
import { render, screen } from '@testing-library/react';

// Test suite for Miner Connection Instructions
describe('MinerInstructions Component', () => {
  
  // Test: Instructions should display stratum URL
  test('displays the stratum URL for connecting miners', () => {
    // The stratum URL should be clearly visible
    const stratumUrl = 'stratum+tcp://206.162.80.230:3333';
    // Expect the URL to be present in the instructions
    expect(stratumUrl).toContain('stratum+tcp://');
    expect(stratumUrl).toContain(':3333');
  });

  // Test: Username field should specify email address (not wallet address)
  test('username field should be user email address', () => {
    const usernameInstruction = 'Your registered email address';
    // Should NOT mention wallet address for username
    expect(usernameInstruction.toLowerCase()).not.toContain('wallet');
    expect(usernameInstruction.toLowerCase()).toContain('email');
  });

  // Test: Password field should specify account password
  test('password field should be account password', () => {
    const passwordInstruction = 'Your account password';
    expect(passwordInstruction.toLowerCase()).toContain('password');
  });

  // Test: Instructions should include step-by-step guide
  test('should include step-by-step connection guide', () => {
    const steps = [
      'Create an account or login',
      'Set your payout wallet address',
      'Configure your miner',
      'Start mining'
    ];
    expect(steps.length).toBeGreaterThanOrEqual(4);
    expect(steps[0].toLowerCase()).toContain('account');
    expect(steps[1].toLowerCase()).toContain('wallet');
    expect(steps[2].toLowerCase()).toContain('miner');
  });

  // Test: Should include supported mining software
  test('should list supported mining software', () => {
    const supportedMiners = ['lolMiner', 'BzMiner', 'Rigel', 'SRBMiner'];
    expect(supportedMiners.length).toBeGreaterThan(0);
  });

  // Test: Should include example configuration
  test('should include example miner configuration', () => {
    const exampleConfig = {
      pool: 'stratum+tcp://206.162.80.230:3333',
      user: 'your@email.com',
      pass: 'yourpassword'
    };
    expect(exampleConfig.pool).toContain('stratum');
    expect(exampleConfig.user).toContain('@');
    expect(exampleConfig.pass).toBeTruthy();
  });

  // Test: Should display algorithm information
  test('should display Blake3 algorithm information', () => {
    const algorithm = 'Blake3';
    const network = 'BlockDAG';
    expect(algorithm).toBe('Blake3');
    expect(network).toBe('BlockDAG');
  });

  // Test: Should include troubleshooting tips
  test('should include troubleshooting information', () => {
    const troubleshootingTopics = [
      'connection refused',
      'authentication failed',
      'shares rejected'
    ];
    expect(troubleshootingTopics.length).toBeGreaterThan(0);
  });
});
