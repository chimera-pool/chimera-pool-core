import { useState, useCallback, useEffect } from 'react';
import { IUserChartPreferences } from '../interfaces/IChartRegistry';

const STORAGE_KEY = 'chimera-chart-preferences';

/**
 * Default empty preferences
 */
const defaultPreferences: IUserChartPreferences = {
  userId: 'anonymous',
  dashboards: {},
  updatedAt: new Date(),
};

/**
 * Load preferences from localStorage
 */
function loadPreferences(): IUserChartPreferences {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      return {
        ...parsed,
        updatedAt: new Date(parsed.updatedAt),
      };
    }
  } catch (error) {
    console.warn('Failed to load chart preferences:', error);
  }
  return defaultPreferences;
}

/**
 * Save preferences to localStorage
 */
function savePreferences(prefs: IUserChartPreferences): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch (error) {
    console.warn('Failed to save chart preferences:', error);
  }
}

/**
 * Hook for managing user chart preferences
 */
export function useChartPreferences(userId?: string) {
  const [preferences, setPreferences] = useState<IUserChartPreferences>(() => {
    const loaded = loadPreferences();
    if (userId && loaded.userId !== userId) {
      return { ...defaultPreferences, userId };
    }
    return loaded;
  });

  // Update userId if it changes
  useEffect(() => {
    if (userId && preferences.userId !== userId) {
      setPreferences(prev => ({ ...prev, userId }));
    }
  }, [userId, preferences.userId]);

  /**
   * Get the selected chart for a specific slot
   */
  const getSlotSelection = useCallback(
    (dashboardId: string, slotId: string): string | undefined => {
      return preferences.dashboards[dashboardId]?.[slotId];
    },
    [preferences]
  );

  /**
   * Set the selected chart for a specific slot
   */
  const setSlotSelection = useCallback(
    (dashboardId: string, slotId: string, chartId: string) => {
      setPreferences(prev => {
        const updated: IUserChartPreferences = {
          ...prev,
          dashboards: {
            ...prev.dashboards,
            [dashboardId]: {
              ...prev.dashboards[dashboardId],
              [slotId]: chartId,
            },
          },
          updatedAt: new Date(),
        };
        savePreferences(updated);
        return updated;
      });
    },
    []
  );

  /**
   * Reset all preferences to defaults
   */
  const resetPreferences = useCallback(() => {
    const reset = { ...defaultPreferences, userId: preferences.userId };
    setPreferences(reset);
    savePreferences(reset);
  }, [preferences.userId]);

  /**
   * Reset preferences for a specific dashboard
   */
  const resetDashboard = useCallback(
    (dashboardId: string) => {
      setPreferences(prev => {
        const { [dashboardId]: _, ...otherDashboards } = prev.dashboards;
        const updated: IUserChartPreferences = {
          ...prev,
          dashboards: otherDashboards,
          updatedAt: new Date(),
        };
        savePreferences(updated);
        return updated;
      });
    },
    []
  );

  return {
    preferences,
    getSlotSelection,
    setSlotSelection,
    resetPreferences,
    resetDashboard,
  };
}

export default useChartPreferences;
