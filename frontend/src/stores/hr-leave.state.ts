import { defineStore } from 'pinia';

import {
  leaveService,
  type LeaveRequest,
  type ListLeaveRequestsResponse,
  type GetCalendarEventsResponse,
} from '../api/client';

export const useHrLeaveStore = defineStore('hr-leave', () => {
  async function listLeaveRequests(
    paging?: { page?: number; pageSize?: number },
    formValues?: {
      userId?: number;
      absenceTypeId?: string;
      status?: string;
      startDate?: string;
      endDate?: string;
    } | null,
  ): Promise<ListLeaveRequestsResponse> {
    return await leaveService.ListLeaveRequests({
      userId: formValues?.userId,
      absenceTypeId: formValues?.absenceTypeId,
      status: formValues?.status as any,
      startDate: formValues?.startDate,
      endDate: formValues?.endDate,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getLeaveRequest(id: string) {
    return await leaveService.GetLeaveRequest({ id });
  }

  async function createLeaveRequest(data: Partial<LeaveRequest>) {
    return await leaveService.CreateLeaveRequest(data as any);
  }

  async function updateLeaveRequest(
    id: string,
    data: Partial<LeaveRequest>,
    updateMask: string[],
  ) {
    return await leaveService.UpdateLeaveRequest({
      id,
      data: data as LeaveRequest,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteLeaveRequest(id: string) {
    return await leaveService.DeleteLeaveRequest({ id });
  }

  async function approveLeaveRequest(
    id: string,
    reviewNotes?: string,
    approverEmail?: string,
    approverName?: string,
  ) {
    return await leaveService.ApproveLeaveRequest({
      id,
      reviewNotes,
      approverEmail,
      approverName,
    });
  }

  async function rejectLeaveRequest(id: string, reviewNotes?: string) {
    return await leaveService.RejectLeaveRequest({ id, reviewNotes });
  }

  async function cancelLeaveRequest(id: string) {
    return await leaveService.CancelLeaveRequest({ id });
  }

  async function revokeLeaveRequest(id: string, reason?: string) {
    return await leaveService.RevokeLeaveRequest({ id, reason });
  }

  async function getCalendarEvents(params?: {
    startDate?: string;
    endDate?: string;
    orgUnitName?: string;
    userId?: number;
  }): Promise<GetCalendarEventsResponse> {
    return await leaveService.GetCalendarEvents(params ?? {});
  }

  function $reset() {}

  return {
    $reset,
    listLeaveRequests,
    getLeaveRequest,
    createLeaveRequest,
    updateLeaveRequest,
    deleteLeaveRequest,
    approveLeaveRequest,
    rejectLeaveRequest,
    cancelLeaveRequest,
    revokeLeaveRequest,
    getCalendarEvents,
  };
});
