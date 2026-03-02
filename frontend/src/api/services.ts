/**
 * HR Module Service Functions
 *
 * Typed service methods for the HR API using dynamic module routing.
 * Base URL: /admin/v1/modules/hr/v1
 */

import { hrApi, type RequestOptions } from './client';

// Paging utility type
export interface Paging {
  page?: number;
  pageSize?: number;
}

// ==================== Enum Types ====================

export type LeaveRequestStatus =
  | 'LEAVE_REQUEST_STATUS_UNSPECIFIED'
  | 'LEAVE_REQUEST_STATUS_PENDING'
  | 'LEAVE_REQUEST_STATUS_APPROVED'
  | 'LEAVE_REQUEST_STATUS_REJECTED'
  | 'LEAVE_REQUEST_STATUS_CANCELLED'
  | 'LEAVE_REQUEST_STATUS_AWAITING_SIGNING';

// ==================== Entity Types ====================

export interface AbsenceType {
  id: string;
  tenantId?: number;
  name: string;
  description?: string;
  color?: string;
  icon?: string;
  deductsFromAllowance?: boolean;
  requiresApproval?: boolean;
  isActive?: boolean;
  sortOrder?: number;
  metadata?: Record<string, unknown>;
  requiresSigning?: boolean;
  signingTemplateId?: string;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: number;
  updatedBy?: number;
}

export interface LeaveRequest {
  id: string;
  tenantId?: number;
  userId?: number;
  absenceTypeId?: string;
  startDate?: string;
  endDate?: string;
  days?: number;
  status?: LeaveRequestStatus;
  reason?: string;
  reviewNotes?: string;
  reviewedBy?: number;
  reviewedAt?: string;
  notes?: string;
  metadata?: Record<string, unknown>;
  userName?: string;
  userEmail?: string;
  absenceTypeName?: string;
  absenceTypeColor?: string;
  reviewerName?: string;
  signingRequestId?: string;
  orgUnitName?: string;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: number;
  updatedBy?: number;
}

export interface LeaveAllowance {
  id: string;
  tenantId?: number;
  userId?: number;
  absenceTypeId?: string;
  year?: number;
  totalDays?: number;
  usedDays?: number;
  carriedOver?: number;
  notes?: string;
  userName?: string;
  absenceTypeName?: string;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: number;
  updatedBy?: number;
}

export interface CalendarEvent {
  id: string;
  userId?: number;
  userName?: string;
  absenceTypeId?: string;
  absenceTypeName?: string;
  color?: string;
  startDate?: string;
  endDate?: string;
  days?: number;
  status?: LeaveRequestStatus;
  orgUnitName?: string;
}

export interface BalanceEntry {
  absenceTypeId: string;
  absenceTypeName?: string;
  color?: string;
  totalDays: number;
  usedDays: number;
  carriedOver: number;
  remainingDays: number;
}

export interface HealthCheckResponse {
  status: string;
  version?: string;
  timestamp?: string;
}

export interface HrStats {
  pendingRequests?: number;
  approvedRequestsThisMonth?: number;
  activeAbsenceTypes?: number;
}

// ==================== Response Types ====================

export interface ListAbsenceTypesResponse {
  items: AbsenceType[];
  total: number;
}

export interface ListLeaveRequestsResponse {
  items: LeaveRequest[];
  total: number;
}

export interface ListAllowancesResponse {
  items: LeaveAllowance[];
  total: number;
}

export interface GetCalendarEventsResponse {
  events: CalendarEvent[];
}

export interface GetUserBalanceResponse {
  userId: number;
  year: number;
  entries: BalanceEntry[];
}

// ==================== Helper ====================

/** Convert a YYYY-MM-DD date string to RFC 3339 timestamp for protobuf Timestamp fields */
export function toTimestamp(date?: string): string | undefined {
  if (!date) return undefined;
  if (date.includes('T')) return date;
  return `${date}T00:00:00Z`;
}

/** Extract YYYY-MM-DD from an RFC 3339 timestamp (for form inputs) */
export function fromTimestamp(ts?: string): string {
  if (!ts) return '';
  return ts.split('T')[0] || '';
}

function buildQuery(params: Record<string, unknown>): string {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      if (Array.isArray(value)) {
        value.forEach((v) => searchParams.append(key, String(v)));
      } else {
        searchParams.append(key, String(value));
      }
    }
  }
  const query = searchParams.toString();
  return query ? `?${query}` : '';
}

// ==================== System Service ====================

export const SystemService = {
  healthCheck: async (
    options?: RequestOptions,
  ): Promise<HealthCheckResponse> => {
    return hrApi.get<HealthCheckResponse>('/health', options);
  },

  getStats: async (options?: RequestOptions): Promise<{ stats: HrStats }> => {
    return hrApi.get<{ stats: HrStats }>('/stats', options);
  },
};

// ==================== Absence Type Service ====================

export const AbsenceTypeService = {
  list: async (
    params?: {
      query?: string;
      page?: number;
      pageSize?: number;
      noPaging?: boolean;
    },
    options?: RequestOptions,
  ): Promise<ListAbsenceTypesResponse> => {
    return hrApi.get<ListAbsenceTypesResponse>(
      `/absence-types${buildQuery(params || {})}`,
      options,
    );
  },

  get: async (
    id: string,
    options?: RequestOptions,
  ): Promise<{ absenceType: AbsenceType }> => {
    return hrApi.get<{ absenceType: AbsenceType }>(
      `/absence-types/${id}`,
      options,
    );
  },

  create: async (
    data: Partial<AbsenceType>,
    options?: RequestOptions,
  ): Promise<{ absenceType: AbsenceType }> => {
    return hrApi.post<{ absenceType: AbsenceType }>(
      '/absence-types',
      data,
      options,
    );
  },

  update: async (
    id: string,
    data: { id: string; data: AbsenceType; updateMask: string },
    options?: RequestOptions,
  ): Promise<{ absenceType: AbsenceType }> => {
    return hrApi.put<{ absenceType: AbsenceType }>(
      `/absence-types/${id}`,
      data,
      options,
    );
  },

  delete: async (id: string, options?: RequestOptions): Promise<void> => {
    return hrApi.delete(`/absence-types/${id}`, options);
  },
};

// ==================== Leave Request Service ====================

export const LeaveService = {
  list: async (
    params?: {
      userId?: number;
      status?: string;
      startDate?: string;
      endDate?: string;
      page?: number;
      pageSize?: number;
      noPaging?: boolean;
    },
    options?: RequestOptions,
  ): Promise<ListLeaveRequestsResponse> => {
    return hrApi.get<ListLeaveRequestsResponse>(
      `/leave-requests${buildQuery(params || {})}`,
      options,
    );
  },

  get: async (
    id: string,
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.get<{ leaveRequest: LeaveRequest }>(
      `/leave-requests/${id}`,
      options,
    );
  },

  create: async (
    data: Partial<LeaveRequest>,
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.post<{ leaveRequest: LeaveRequest }>(
      '/leave-requests',
      data,
      options,
    );
  },

  update: async (
    id: string,
    data: { id: string; data: LeaveRequest; updateMask: string },
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.put<{ leaveRequest: LeaveRequest }>(
      `/leave-requests/${id}`,
      data,
      options,
    );
  },

  delete: async (id: string, options?: RequestOptions): Promise<void> => {
    return hrApi.delete(`/leave-requests/${id}`, options);
  },

  approve: async (
    id: string,
    data?: {
      reviewNotes?: string;
      approverEmail?: string;
      approverName?: string;
    },
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.post<{ leaveRequest: LeaveRequest }>(
      `/leave-requests/${id}/approve`,
      data,
      options,
    );
  },

  reject: async (
    id: string,
    data?: { reviewNotes?: string },
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.post<{ leaveRequest: LeaveRequest }>(
      `/leave-requests/${id}/reject`,
      data,
      options,
    );
  },

  cancel: async (
    id: string,
    options?: RequestOptions,
  ): Promise<{ leaveRequest: LeaveRequest }> => {
    return hrApi.post<{ leaveRequest: LeaveRequest }>(
      `/leave-requests/${id}/cancel`,
      undefined,
      options,
    );
  },

  getCalendarEvents: async (
    params?: {
      startDate?: string;
      endDate?: string;
      orgUnitName?: string;
      userId?: number;
    },
    options?: RequestOptions,
  ): Promise<GetCalendarEventsResponse> => {
    return hrApi.get<GetCalendarEventsResponse>(
      `/calendar${buildQuery(params || {})}`,
      options,
    );
  },
};

// ==================== Allowance Service ====================

export const AllowanceService = {
  list: async (
    params?: {
      userId?: number;
      absenceTypeId?: string;
      year?: number;
      page?: number;
      pageSize?: number;
      noPaging?: boolean;
    },
    options?: RequestOptions,
  ): Promise<ListAllowancesResponse> => {
    return hrApi.get<ListAllowancesResponse>(
      `/allowances${buildQuery(params || {})}`,
      options,
    );
  },

  get: async (
    id: string,
    options?: RequestOptions,
  ): Promise<{ allowance: LeaveAllowance }> => {
    return hrApi.get<{ allowance: LeaveAllowance }>(
      `/allowances/${id}`,
      options,
    );
  },

  create: async (
    data: Partial<LeaveAllowance>,
    options?: RequestOptions,
  ): Promise<{ allowance: LeaveAllowance }> => {
    return hrApi.post<{ allowance: LeaveAllowance }>(
      '/allowances',
      data,
      options,
    );
  },

  update: async (
    id: string,
    data: { id: string; data: LeaveAllowance; updateMask: string },
    options?: RequestOptions,
  ): Promise<{ allowance: LeaveAllowance }> => {
    return hrApi.put<{ allowance: LeaveAllowance }>(
      `/allowances/${id}`,
      data,
      options,
    );
  },

  delete: async (id: string, options?: RequestOptions): Promise<void> => {
    return hrApi.delete(`/allowances/${id}`, options);
  },

  getUserBalance: async (
    userId: number,
    year?: number,
    options?: RequestOptions,
  ): Promise<GetUserBalanceResponse> => {
    const params: Record<string, unknown> = {};
    if (year) params.year = year;
    return hrApi.get<GetUserBalanceResponse>(
      `/users/${userId}/balance${buildQuery(params)}`,
      options,
    );
  },
};
