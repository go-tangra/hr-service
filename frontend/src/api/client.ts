/**
 * Auto-generated API client for HR module
 *
 * This client uses the dynamic module routing:
 *   /admin/v1/modules/hr/v1/...
 */

import { useAccessStore } from 'shell/vben/stores';

const MODULE_BASE_URL = '/admin/v1/modules/hr/v1';

export interface RequestOptions {
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

function getAuthHeaders(): Record<string, string> {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  options?: RequestOptions & { baseUrl?: string },
): Promise<T> {
  const base = options?.baseUrl ?? MODULE_BASE_URL;
  const url = `${base}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
      ...options?.headers,
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: options?.signal,
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  return response.json();
}

export const hrApi = {
  get: <T>(path: string, options?: RequestOptions) =>
    request<T>('GET', path, undefined, options),

  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('POST', path, body, options),

  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('PUT', path, body, options),

  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('PATCH', path, body, options),

  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>('DELETE', path, undefined, options),
};

/** Client for admin/portal API calls (e.g. /admin/v1/users) */
export const adminApi = {
  get: <T>(path: string, params?: Record<string, unknown>) => {
    const query = params
      ? '?' + new URLSearchParams(
          Object.entries(params)
            .filter(([, v]) => v != null)
            .map(([k, v]) => [k, String(v)]),
        ).toString()
      : '';
    return request<T>('GET', `${path}${query}`, undefined, { baseUrl: '/admin/admin/v1' });
  },
};

export default hrApi;
