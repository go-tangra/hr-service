import { defineStore } from 'pinia';

import { systemService, type GetStatsResponse } from '../api/client';

export const useHrSystemStore = defineStore('hr-system', () => {
  async function getStats(): Promise<GetStatsResponse> {
    return await systemService.GetStats({});
  }

  function $reset() {}

  return {
    $reset,
    getStats,
  };
});
