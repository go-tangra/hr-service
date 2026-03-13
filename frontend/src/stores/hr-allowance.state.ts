import { defineStore } from 'pinia';

import {
  allowanceService,
  type LeaveAllowance,
  type ListAllowancesResponse,
  type GetUserBalanceResponse,
} from '../api/client';

export const useHrAllowanceStore = defineStore('hr-allowance', () => {
  async function listAllowances(
    paging?: { page?: number; pageSize?: number },
    formValues?: {
      userId?: number;
      absenceTypeId?: string;
      year?: number;
    } | null,
  ): Promise<ListAllowancesResponse> {
    return await allowanceService.ListAllowances({
      userId: formValues?.userId,
      absenceTypeId: formValues?.absenceTypeId,
      year: formValues?.year,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getAllowance(id: string) {
    return await allowanceService.GetAllowance({ id });
  }

  async function createAllowance(data: Partial<LeaveAllowance>) {
    return await allowanceService.CreateAllowance(data as any);
  }

  async function updateAllowance(
    id: string,
    data: Partial<LeaveAllowance>,
    updateMask: string[],
  ) {
    return await allowanceService.UpdateAllowance({
      id,
      data: data as LeaveAllowance,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteAllowance(id: string) {
    return await allowanceService.DeleteAllowance({ id });
  }

  async function getUserBalance(
    userId: number,
    year?: number,
  ): Promise<GetUserBalanceResponse> {
    return await allowanceService.GetUserBalance({ userId, year });
  }

  function $reset() {}

  return {
    $reset,
    listAllowances,
    getAllowance,
    createAllowance,
    updateAllowance,
    deleteAllowance,
    getUserBalance,
  };
});
