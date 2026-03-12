import { computed } from 'vue';
import { useAccessStore, useUserStore } from 'shell/vben/stores';

export function usePermission() {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const isAdmin = computed(() => userStore.userRoles?.includes('platform:admin') || userStore.userRoles?.includes('tenant:manager'));
  const can = (code: string) => isAdmin.value || accessStore.accessCodes.includes(code);

  return {
    canManageRequests: computed(() => can('hr.request.manage')),
    canDeleteRequests: computed(() => can('hr.request.delete')),
    canApproveRequests: computed(() => can('hr.request.approve')),
    canManageAllowances: computed(() => can('hr.allowance.manage')),
    canManageAbsenceTypes: computed(() => can('hr.absence_type.manage')),
  };
}
