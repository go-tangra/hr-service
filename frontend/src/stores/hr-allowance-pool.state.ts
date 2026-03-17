import { defineStore } from 'pinia';

import {
  allowancePoolService,
  type AllowancePool,
  type ListAllowancePoolsResponse,
} from '../api/client';

export const useHrAllowancePoolStore = defineStore('hr-allowance-pool', () => {
  async function listAllowancePools(
    paging?: { page?: number; pageSize?: number },
    formValues?: { query?: string } | null,
  ): Promise<ListAllowancePoolsResponse> {
    return await allowancePoolService.ListAllowancePools({
      query: formValues?.query,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getAllowancePool(id: string) {
    return await allowancePoolService.GetAllowancePool({ id });
  }

  async function createAllowancePool(data: Partial<AllowancePool> & { absenceTypeIds?: string[] }) {
    return await allowancePoolService.CreateAllowancePool(data as any);
  }

  async function updateAllowancePool(
    id: string,
    data: Partial<AllowancePool>,
    updateMask: string[],
  ) {
    return await allowancePoolService.UpdateAllowancePool({
      id,
      data: data as AllowancePool,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteAllowancePool(id: string) {
    return await allowancePoolService.DeleteAllowancePool({ id });
  }

  function $reset() {}

  return {
    $reset,
    listAllowancePools,
    getAllowancePool,
    createAllowancePool,
    updateAllowancePool,
    deleteAllowancePool,
  };
});
