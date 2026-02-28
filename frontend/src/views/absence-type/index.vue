<script lang="ts" setup>
import { h } from 'vue';

import { Page, useVbenModal, type VbenFormProps } from 'shell/vben/common-ui';
import { LucideEye, LucideTrash, LucidePencil } from 'shell/vben/icons';

import { notification, Space, Button, Tag } from 'ant-design-vue';

import { useVbenVxeGrid } from 'shell/adapter/vxe-table';
import type { VxeGridProps } from 'shell/adapter/vxe-table';
import type { AbsenceType } from '../../api/services';
import { $t } from 'shell/locales';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';

import AbsenceTypeDrawer from './absence-type-drawer.vue';

const absenceTypeStore = useHrAbsenceTypeStore();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'query',
      label: $t('ui.table.search'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<AbsenceType> = {
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
        const resp = await absenceTypeStore.listAbsenceTypes(
          { page: page.currentPage, pageSize: page.pageSize },
          {
            query: formValues?.query,
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
      title: $t('hr.page.absenceType.name'),
      field: 'name',
      minWidth: 150,
      slots: { default: 'nameSlot' },
    },
    {
      title: $t('hr.page.absenceType.description'),
      field: 'description',
      minWidth: 200,
    },
    {
      title: $t('hr.page.absenceType.deductsFromAllowance'),
      field: 'deductsFromAllowance',
      width: 160,
      slots: { default: 'deducts' },
    },
    {
      title: $t('hr.page.absenceType.requiresApproval'),
      field: 'requiresApproval',
      width: 150,
      slots: { default: 'approval' },
    },
    {
      title: $t('hr.page.absenceType.isActive'),
      field: 'isActive',
      width: 100,
      slots: { default: 'active' },
    },
    {
      title: $t('hr.page.absenceType.sortOrder'),
      field: 'sortOrder',
      width: 100,
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

const [AbsenceTypeDrawerComponent, absenceTypeDrawerApi] = useVbenModal({
  connectedComponent: AbsenceTypeDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.query();
    }
  },
});

function openModal(row: AbsenceType, mode: 'create' | 'edit' | 'view') {
  absenceTypeDrawerApi.setData({ row, mode });
  absenceTypeDrawerApi.open();
}

function handleView(row: AbsenceType) {
  openModal(row, 'view');
}

function handleEdit(row: AbsenceType) {
  openModal(row, 'edit');
}

function handleCreate() {
  openModal({} as AbsenceType, 'create');
}

async function handleDelete(row: AbsenceType) {
  if (!row.id) return;
  try {
    await absenceTypeStore.deleteAbsenceType(row.id);
    notification.success({
      message: $t('hr.page.absenceType.deleteSuccess'),
    });
    await gridApi.query();
  } catch {
    notification.error({ message: $t('ui.notification.delete_failed') });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('hr.page.absenceType.title')">
      <template #toolbar-tools>
        <Button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('hr.page.absenceType.create') }}
        </Button>
      </template>
      <template #nameSlot="{ row }">
        <span class="flex items-center gap-2">
          <span
            v-if="row.color"
            class="inline-block h-3 w-3 rounded-full"
            :style="{ backgroundColor: row.color }"
          />
          {{ row.name }}
        </span>
      </template>
      <template #deducts="{ row }">
        <Tag :color="row.deductsFromAllowance ? 'blue' : 'default'">
          {{ row.deductsFromAllowance ? 'Yes' : 'No' }}
        </Tag>
      </template>
      <template #approval="{ row }">
        <Tag :color="row.requiresApproval ? 'orange' : 'default'">
          {{ row.requiresApproval ? 'Yes' : 'No' }}
        </Tag>
      </template>
      <template #active="{ row }">
        <Tag :color="row.isActive ? 'green' : 'red'">
          {{ row.isActive ? 'Yes' : 'No' }}
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
            type="link"
            size="small"
            :icon="h(LucidePencil)"
            :title="$t('ui.button.edit')"
            @click.stop="handleEdit(row)"
          />
          <a-popconfirm
            :cancel-text="$t('ui.button.cancel')"
            :ok-text="$t('ui.button.ok')"
            :title="$t('hr.page.absenceType.confirmDelete')"
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

    <AbsenceTypeDrawerComponent />
  </Page>
</template>
