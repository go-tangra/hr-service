/**
 * HR Module API Client
 *
 * Uses buf-generated TypeScript clients from protoc-gen-typescript-http.
 * All types and service methods are auto-generated from protos.
 */

import { useAccessStore } from 'shell/vben/stores';

import {
  createHrAbsenceTypeServiceClient,
  createHrAllowanceServiceClient,
  createHrAllowancePoolServiceClient,
  createHrLeaveServiceClient,
  createHrSystemServiceClient,
  createHrUserServiceClient,
} from '../generated/api/hr/service/v1';

const MODULE_BASE_URL = '/admin/v1/modules/hr';

type RequestType = {
  path: string;
  method: string;
  body: string | null;
};

async function handler(req: RequestType): Promise<unknown> {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;

  const response = await fetch(`${MODULE_BASE_URL}/${req.path}`, {
    method: req.method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: req.body,
  });

  if (!response.ok) {
    let message = `HTTP error! status: ${response.status}`;
    try {
      const errorBody = await response.json();
      if (errorBody?.message) {
        message = errorBody.message;
      }
    } catch { /* response body not JSON, use default message */ }
    throw new Error(message);
  }

  return response.json();
}

// Generated typed service clients
export const absenceTypeService = createHrAbsenceTypeServiceClient(handler);
export const allowanceService = createHrAllowanceServiceClient(handler);
export const allowancePoolService = createHrAllowancePoolServiceClient(handler);
export const leaveService = createHrLeaveServiceClient(handler);
export const systemService = createHrSystemServiceClient(handler);
export const userService = createHrUserServiceClient(handler);

// Re-export all generated types for convenience
export type {
  AbsenceType,
  LeaveRequest,
  LeaveAllowance,
  LeaveRequestStatus,
  CalendarEvent,
  BalanceEntry,
  AllowancePool,
  HrUser,
  ListAbsenceTypesResponse,
  ListLeaveRequestsResponse,
  ListAllowancesResponse,
  ListAllowancePoolsResponse,
  GetCalendarEventsResponse,
  GetUserBalanceResponse,
  GetStatsResponse,
  HrErrorReason,
} from '../generated/api/hr/service/v1';

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

/** Client for paperless module API calls (different base URL) */
export const paperlessApi = {
  get: async <T>(path: string, params?: Record<string, unknown>): Promise<T> => {
    const query = params
      ? '?' + new URLSearchParams(
          Object.entries(params)
            .filter(([, v]) => v != null)
            .map(([k, v]) => [k, String(v)]),
        ).toString()
      : '';
    const accessStore = useAccessStore();
    const token = accessStore.accessToken;
    const response = await fetch(`/admin/v1/modules/paperless/v1${path}${query}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
    if (!response.ok) {
      let message = `HTTP error! status: ${response.status}`;
      try {
        const errorBody = await response.json();
        if (errorBody?.message) message = errorBody.message;
      } catch {}
      throw new Error(message);
    }
    return response.json();
  },
};
