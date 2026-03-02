<script lang="ts" setup>
import { h } from 'vue';

import { Page, useVbenModal, type VbenFormProps } from 'shell/vben/common-ui';
import { LucideEye, LucideTrash, LucideCheck, LucideX, LucideFileDown } from 'shell/vben/icons';

import { notification, Space, Button, Tag, Tooltip } from 'ant-design-vue';

import { useVbenVxeGrid } from 'shell/adapter/vxe-table';
import type { VxeGridProps } from 'shell/adapter/vxe-table';
import type { LeaveRequest } from '../../api/services';
import { fromTimestamp } from '../../api/services';
import { $t } from 'shell/locales';
import { useHrLeaveStore } from '../../stores/hr-leave.state';
import { paperlessApi } from '../../api/client';

import RequestDrawer from './request-drawer.vue';
import ReviewModal from './review-modal.vue';

const leaveStore = useHrLeaveStore();

function statusColor(status?: string): string {
  switch (status) {
    case 'LEAVE_REQUEST_STATUS_APPROVED': return 'green';
    case 'LEAVE_REQUEST_STATUS_REJECTED': return 'red';
    case 'LEAVE_REQUEST_STATUS_CANCELLED': return 'default';
    case 'LEAVE_REQUEST_STATUS_PENDING': return 'orange';
    case 'LEAVE_REQUEST_STATUS_AWAITING_SIGNING': return 'blue';
    default: return 'default';
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case 'LEAVE_REQUEST_STATUS_APPROVED': return $t('hr.enum.leaveRequestStatus.approved');
    case 'LEAVE_REQUEST_STATUS_REJECTED': return $t('hr.enum.leaveRequestStatus.rejected');
    case 'LEAVE_REQUEST_STATUS_CANCELLED': return $t('hr.enum.leaveRequestStatus.cancelled');
    case 'LEAVE_REQUEST_STATUS_PENDING': return $t('hr.enum.leaveRequestStatus.pending');
    case 'LEAVE_REQUEST_STATUS_AWAITING_SIGNING': return $t('hr.enum.leaveRequestStatus.awaitingSigning');
    default: return '';
  }
}

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('hr.page.request.status'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        options: [
          { label: $t('hr.enum.leaveRequestStatus.pending'), value: 'LEAVE_REQUEST_STATUS_PENDING' },
          { label: $t('hr.enum.leaveRequestStatus.approved'), value: 'LEAVE_REQUEST_STATUS_APPROVED' },
          { label: $t('hr.enum.leaveRequestStatus.rejected'), value: 'LEAVE_REQUEST_STATUS_REJECTED' },
          { label: $t('hr.enum.leaveRequestStatus.cancelled'), value: 'LEAVE_REQUEST_STATUS_CANCELLED' },
          { label: $t('hr.enum.leaveRequestStatus.awaitingSigning'), value: 'LEAVE_REQUEST_STATUS_AWAITING_SIGNING' },
        ],
      },
    },
  ],
};

const gridOptions: VxeGridProps<LeaveRequest> = {
  height: 'auto',
  stripe: false,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  rowConfig: {
    isHover: true,
  },
  pagerConfig: {
    enabled: true,
    pageSize: 20,
    pageSizes: [10, 20, 50, 100],
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        const resp = await leaveStore.listLeaveRequests(
          { page: page.currentPage, pageSize: page.pageSize },
          {
            status: formValues?.status,
          },
        );
        return {
          items: resp.items ?? [],
          total: resp.total ?? 0,
        };
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    {
      title: $t('hr.page.request.userName'),
      field: 'userName',
      minWidth: 150,
    },
    {
      title: $t('hr.page.request.absenceTypeName'),
      field: 'absenceTypeName',
      width: 140,
    },
    {
      title: $t('hr.page.request.startDate'),
      field: 'startDate',
      width: 120,
      formatter: ({ cellValue }: { cellValue: string }) => fromTimestamp(cellValue),
    },
    {
      title: $t('hr.page.request.endDate'),
      field: 'endDate',
      width: 120,
      formatter: ({ cellValue }: { cellValue: string }) => fromTimestamp(cellValue),
    },
    {
      title: $t('hr.page.request.days'),
      field: 'days',
      width: 80,
    },
    {
      title: $t('hr.page.request.status'),
      field: 'status',
      width: 120,
      slots: { default: 'status' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 180,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [RequestDrawerComponent, requestDrawerApi] = useVbenModal({
  connectedComponent: RequestDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.query();
    }
  },
});

const [ReviewModalComponent, reviewModalApi] = useVbenModal({
  connectedComponent: ReviewModal,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.query();
    }
  },
});

function openModal(row: LeaveRequest, mode: 'create' | 'edit' | 'view') {
  requestDrawerApi.setData({ row, mode });
  requestDrawerApi.open();
}

function handleView(row: LeaveRequest) {
  openModal(row, 'view');
}

function handleCreate() {
  openModal({} as LeaveRequest, 'create');
}

function handleApprove(row: LeaveRequest) {
  reviewModalApi.setData({ action: 'approve', requestId: row.id, row });
  reviewModalApi.open();
}

function handleReject(row: LeaveRequest) {
  reviewModalApi.setData({ action: 'reject', requestId: row.id, row });
  reviewModalApi.open();
}

async function handleDownloadSigned(row: LeaveRequest) {
  if (!row.signingRequestId) return;
  try {
    const resp = await paperlessApi.get<{ url: string }>(`/signing/requests/${row.signingRequestId}/download`);
    if (resp.url) {
      window.open(resp.url, '_blank');
    }
  } catch {
    notification.error({ message: $t('hr.page.request.downloadFailed') });
  }
}

async function handleDelete(row: LeaveRequest) {
  if (!row.id) return;
  try {
    await leaveStore.deleteLeaveRequest(row.id);
    notification.success({
      message: $t('hr.page.request.deleteSuccess'),
    });
    await gridApi.query();
  } catch {
    notification.error({ message: $t('ui.notification.delete_failed') });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('hr.page.request.title')">
      <template #toolbar-tools>
        <Button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('hr.page.request.create') }}
        </Button>
      </template>
      <template #status="{ row }">
        <Tag :color="statusColor(row.status)">
          {{ statusLabel(row.status) }}
        </Tag>
      </template>
      <template #action="{ row }">
        <Space>
          <Button
            type="link"
            size="small"
            :icon="h(LucideEye)"
            :title="$t('ui.button.view')"
            @click.stop="handleView(row)"
          />
          <Button
            v-if="row.status === 'LEAVE_REQUEST_STATUS_PENDING'"
            type="link"
            size="small"
            :icon="h(LucideCheck)"
            :title="$t('hr.page.request.approve')"
            style="color: #52c41a"
            @click.stop="handleApprove(row)"
          />
          <Button
            v-if="row.status === 'LEAVE_REQUEST_STATUS_PENDING'"
            type="link"
            size="small"
            :icon="h(LucideX)"
            :title="$t('hr.page.request.reject')"
            style="color: #ff4d4f"
            @click.stop="handleReject(row)"
          />
          <Tooltip v-if="row.status === 'LEAVE_REQUEST_STATUS_APPROVED' && row.signingRequestId" :title="$t('hr.page.request.downloadSigned')">
            <Button
              type="link"
              size="small"
              :icon="h(LucideFileDown)"
              style="color: #1890ff"
              @click.stop="handleDownloadSigned(row)"
            />
          </Tooltip>
          <a-popconfirm
            :cancel-text="$t('ui.button.cancel')"
            :ok-text="$t('ui.button.ok')"
            :title="$t('hr.page.request.confirmDelete')"
            @confirm="handleDelete(row)"
          >
            <Button
              danger
              type="link"
              size="small"
              :icon="h(LucideTrash)"
              :title="$t('ui.button.delete', { moduleName: '' })"
            />
          </a-popconfirm>
        </Space>
      </template>
    </Grid>

    <RequestDrawerComponent />
    <ReviewModalComponent />
  </Page>
</template>
