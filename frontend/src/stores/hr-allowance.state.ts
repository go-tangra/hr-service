import { defineStore } from 'pinia';

import {
  AllowanceService,
  type LeaveAllowance,
  type ListAllowancesResponse,
  type GetUserBalanceResponse,
} from '../api/services';
import type { Paging } from '../api/services';

export const useHrAllowanceStore = defineStore('hr-allowance', () => {
  async function listAllowances(
    paging?: Paging,
    formValues?: {
      userId?: number;
      absenceTypeId?: string;
      year?: number;
    } | null,
  ): Promise<ListAllowancesResponse> {
    return await AllowanceService.list({
      userId: formValues?.userId,
      absenceTypeId: formValues?.absenceTypeId,
      year: formValues?.year,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getAllowance(
    id: string,
  ): Promise<{ allowance: LeaveAllowance }> {
    return await AllowanceService.get(id);
  }

  async function createAllowance(
    data: Partial<LeaveAllowance>,
  ): Promise<{ allowance: LeaveAllowance }> {
    return await AllowanceService.create(data);
  }

  async function updateAllowance(
    id: string,
    data: Partial<LeaveAllowance>,
    updateMask: string[],
  ): Promise<{ allowance: LeaveAllowance }> {
    return await AllowanceService.update(id, {
      id,
      data: data as LeaveAllowance,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteAllowance(id: string): Promise<void> {
    return await AllowanceService.delete(id);
  }

  async function getUserBalance(
    userId: number,
    year?: number,
  ): Promise<GetUserBalanceResponse> {
    return await AllowanceService.getUserBalance(userId, year);
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
