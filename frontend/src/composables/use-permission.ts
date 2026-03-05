import { computed } from 'vue';
import { useAccessStore } from 'shell/vben/stores';

export function usePermission() {
  const accessStore = useAccessStore();
  const can = (code: string) => accessStore.accessCodes.includes(code);

  return {
    canManageRequests: computed(() => can('hr.request.manage')),
    canApproveRequests: computed(() => can('hr.request.approve')),
    canManageAllowances: computed(() => can('hr.allowance.manage')),
    canManageAbsenceTypes: computed(() => can('hr.absence_type.manage')),
  };
}
