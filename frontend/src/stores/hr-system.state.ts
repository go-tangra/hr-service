import { defineStore } from 'pinia';

import { SystemService, type HrStats } from '../api/services';

export const useHrSystemStore = defineStore('hr-system', () => {
  async function getStats(): Promise<{ stats: HrStats }> {
    return await SystemService.getStats();
  }

  function $reset() {}

  return {
    $reset,
    getStats,
  };
});
