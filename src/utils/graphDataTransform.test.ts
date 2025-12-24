import {
  transformHashrateData,
  transformSharesData,
  transformMinersData,
  formatHashrate,
  IRawHashrateData,
  IRawSharesData,
  IRawMinersData,
} from './graphDataTransform';

describe('Graph Data Transformation', () => {
  describe('transformHashrateData', () => {
    it('should handle hashrate field name', () => {
      const input: IRawHashrateData[] = [
        { time: '2025-12-23T10:00:00Z', hashrate: 10000000000000 }, // 10 TH/s
      ];
      
      const result = transformHashrateData(input);
      
      expect(result).toHaveLength(1);
      expect(result[0].hashrate).toBe(10000000000000);
      expect(result[0].hashrateTH).toBe(10);
      expect(result[0].hashrateMH).toBe(10000000);
    });

    it('should handle totalHashrate field name', () => {
      const input: IRawHashrateData[] = [
        { time: '2025-12-23T10:00:00Z', totalHashrate: 50000000000000 }, // 50 TH/s
      ];
      
      const result = transformHashrateData(input);
      
      expect(result).toHaveLength(1);
      expect(result[0].hashrate).toBe(50000000000000);
      expect(result[0].hashrateTH).toBe(50);
    });

    it('should prefer hashrate over totalHashrate when both present', () => {
      const input: IRawHashrateData[] = [
        { time: '2025-12-23T10:00:00Z', hashrate: 10000000000000, totalHashrate: 50000000000000 },
      ];
      
      const result = transformHashrateData(input);
      
      expect(result[0].hashrate).toBe(10000000000000);
    });

    it('should handle empty array', () => {
      expect(transformHashrateData([])).toEqual([]);
    });

    it('should handle null/undefined', () => {
      expect(transformHashrateData(null as any)).toEqual([]);
      expect(transformHashrateData(undefined as any)).toEqual([]);
    });

    it('should handle missing hashrate fields', () => {
      const input: IRawHashrateData[] = [
        { time: '2025-12-23T10:00:00Z' },
      ];
      
      const result = transformHashrateData(input);
      
      expect(result[0].hashrate).toBe(0);
      expect(result[0].hashrateTH).toBe(0);
    });
  });

  describe('transformSharesData', () => {
    it('should transform shares data correctly', () => {
      const input: IRawSharesData[] = [
        { time: '2025-12-23T10:00:00Z', validShares: 950, invalidShares: 50 },
      ];
      
      const result = transformSharesData(input);
      
      expect(result).toHaveLength(1);
      expect(result[0].validShares).toBe(950);
      expect(result[0].invalidShares).toBe(50);
      expect(result[0].totalShares).toBe(1000);
      expect(result[0].acceptanceRate).toBe(95);
    });

    it('should use provided totalShares and acceptanceRate', () => {
      const input: IRawSharesData[] = [
        { 
          time: '2025-12-23T10:00:00Z', 
          validShares: 950, 
          invalidShares: 50,
          totalShares: 1000,
          acceptanceRate: 95.5,
        },
      ];
      
      const result = transformSharesData(input);
      
      expect(result[0].totalShares).toBe(1000);
      expect(result[0].acceptanceRate).toBe(95.5);
    });

    it('should handle zero shares', () => {
      const input: IRawSharesData[] = [
        { time: '2025-12-23T10:00:00Z', validShares: 0, invalidShares: 0 },
      ];
      
      const result = transformSharesData(input);
      
      expect(result[0].acceptanceRate).toBe(0);
    });

    it('should handle empty array', () => {
      expect(transformSharesData([])).toEqual([]);
    });
  });

  describe('transformMinersData', () => {
    it('should transform miners data correctly', () => {
      const input: IRawMinersData[] = [
        { time: '2025-12-23T10:00:00Z', activeMiners: 5, uniqueUsers: 3 },
      ];
      
      const result = transformMinersData(input);
      
      expect(result).toHaveLength(1);
      expect(result[0].activeMiners).toBe(5);
      expect(result[0].uniqueUsers).toBe(3);
    });

    it('should default uniqueUsers to activeMiners when missing', () => {
      const input: IRawMinersData[] = [
        { time: '2025-12-23T10:00:00Z', activeMiners: 5 },
      ];
      
      const result = transformMinersData(input);
      
      expect(result[0].uniqueUsers).toBe(5);
    });
  });

  describe('formatHashrate', () => {
    it('should format PH/s correctly', () => {
      expect(formatHashrate(1.5e15)).toBe('1.50 PH/s');
    });

    it('should format TH/s correctly', () => {
      expect(formatHashrate(10.36e12)).toBe('10.36 TH/s');
    });

    it('should format GH/s correctly', () => {
      expect(formatHashrate(500e9)).toBe('500.00 GH/s');
    });

    it('should format MH/s correctly', () => {
      expect(formatHashrate(150e6)).toBe('150.00 MH/s');
    });

    it('should format KH/s correctly', () => {
      expect(formatHashrate(5000)).toBe('5.00 KH/s');
    });

    it('should format H/s correctly', () => {
      expect(formatHashrate(100)).toBe('100.00 H/s');
    });
  });
});
