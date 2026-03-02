import { defineStore } from 'pinia';

import {
  LeaveService,
  type LeaveRequest,
  type ListLeaveRequestsResponse,
  type GetCalendarEventsResponse,
} from '../api/services';
import type { Paging } from '../api/services';

export const useHrLeaveStore = defineStore('hr-leave', () => {
  async function listLeaveRequests(
    paging?: Paging,
    formValues?: {
      userId?: number;
      status?: string;
      startDate?: string;
      endDate?: string;
    } | null,
  ): Promise<ListLeaveRequestsResponse> {
    return await LeaveService.list({
      userId: formValues?.userId,
      status: formValues?.status,
      startDate: formValues?.startDate,
      endDate: formValues?.endDate,
      page: paging?.page,
      pageSize: paging?.pageSize,
    });
  }

  async function getLeaveRequest(
    id: string,
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.get(id);
  }

  async function createLeaveRequest(
    data: Partial<LeaveRequest>,
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.create(data);
  }

  async function updateLeaveRequest(
    id: string,
    data: Partial<LeaveRequest>,
    updateMask: string[],
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.update(id, {
      id,
      data: data as LeaveRequest,
      updateMask: updateMask.join(','),
    });
  }

  async function deleteLeaveRequest(id: string): Promise<void> {
    return await LeaveService.delete(id);
  }

  async function approveLeaveRequest(
    id: string,
    reviewNotes?: string,
    approverEmail?: string,
    approverName?: string,
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.approve(id, {
      reviewNotes,
      approverEmail,
      approverName,
    });
  }

  async function rejectLeaveRequest(
    id: string,
    reviewNotes?: string,
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.reject(id, { reviewNotes });
  }

  async function cancelLeaveRequest(
    id: string,
  ): Promise<{ leaveRequest: LeaveRequest }> {
    return await LeaveService.cancel(id);
  }

  async function getCalendarEvents(params?: {
    startDate?: string;
    endDate?: string;
    orgUnitName?: string;
    userId?: number;
  }): Promise<GetCalendarEventsResponse> {
    return await LeaveService.getCalendarEvents(params);
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
    getCalendarEvents,
  };
});
