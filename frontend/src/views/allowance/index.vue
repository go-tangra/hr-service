<script lang="ts" setup>
import { h } from 'vue';

import { Page, useVbenModal, type VbenFormProps } from 'shell/vben/common-ui';
import { LucideEye, LucideTrash, LucidePencil } from 'shell/vben/icons';

import { notification, Space, Button } from 'ant-design-vue';

import { useVbenVxeGrid } from 'shell/adapter/vxe-table';
import type { VxeGridProps } from 'shell/adapter/vxe-table';
import type { LeaveAllowance } from '../../api/services';
import { $t } from 'shell/locales';
import { useHrAllowanceStore } from '../../stores/hr-allowance.state';

import AllowanceDrawer from './allowance-drawer.vue';

const allowanceStore = useHrAllowanceStore();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'InputNumber',
      fieldName: 'year',
      label: $t('hr.page.allowance.year'),
      componentProps: {
        placeholder: new Date().getFullYear(),
        min: 2000,
        max: 2099,
        allowClear: true,
        style: { width: '120px' },
      },
    },
  ],
};

const gridOptions: VxeGridProps<LeaveAllowance> = {
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
        const resp = await allowanceStore.listAllowances(
          { page: page.currentPage, pageSize: page.pageSize },
          {
            year: formValues?.year || undefined,
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
      title: $t('hr.page.allowance.userId'),
      field: 'userName',
      minWidth: 150,
    },
    {
      title: $t('hr.page.allowance.absenceTypeId'),
      field: 'absenceTypeName',
      minWidth: 150,
    },
    {
      title: $t('hr.page.allowance.year'),
      field: 'year',
      width: 100,
    },
    {
      title: $t('hr.page.allowance.totalDays'),
      field: 'totalDays',
      width: 120,
    },
    {
      title: $t('hr.page.allowance.usedDays'),
      field: 'usedDays',
      width: 120,
    },
    {
      title: $t('hr.page.allowance.carriedOver'),
      field: 'carriedOver',
      width: 130,
    },
    {
      title: $t('hr.page.allowance.remaining'),
      field: 'remaining',
      width: 130,
      slots: { default: 'remaining' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 150,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [AllowanceDrawerComponent, allowanceDrawerApi] = useVbenModal({
  connectedComponent: AllowanceDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.query();
    }
  },
});

function openModal(row: LeaveAllowance, mode: 'create' | 'edit' | 'view') {
  allowanceDrawerApi.setData({ row, mode });
  allowanceDrawerApi.open();
}

function handleView(row: LeaveAllowance) {
  openModal(row, 'view');
}

function handleEdit(row: LeaveAllowance) {
  openModal(row, 'edit');
}

function handleCreate() {
  openModal({} as LeaveAllowance, 'create');
}

async function handleDelete(row: LeaveAllowance) {
  if (!row.id) return;
  try {
    await allowanceStore.deleteAllowance(row.id);
    notification.success({
      message: $t('hr.page.allowance.deleteSuccess'),
    });
    await gridApi.query();
  } catch {
    notification.error({ message: $t('ui.notification.delete_failed') });
  }
}

function computeRemaining(row: LeaveAllowance): number {
  return (row.totalDays ?? 0) + (row.carriedOver ?? 0) - (row.usedDays ?? 0);
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('hr.page.allowance.title')">
      <template #toolbar-tools>
        <Button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('hr.page.allowance.create') }}
        </Button>
      </template>
      <template #remaining="{ row }">
        <span :style="{ color: computeRemaining(row) <= 0 ? '#f5222d' : undefined, fontWeight: 600 }">
          {{ computeRemaining(row) }}
        </span>
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
            type="link"
            size="small"
            :icon="h(LucidePencil)"
            :title="$t('ui.button.edit')"
            @click.stop="handleEdit(row)"
          />
          <a-popconfirm
            :cancel-text="$t('ui.button.cancel')"
            :ok-text="$t('ui.button.ok')"
            :title="$t('hr.page.allowance.confirmDelete')"
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

    <AllowanceDrawerComponent />
  </Page>
</template>
