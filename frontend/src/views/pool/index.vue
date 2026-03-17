<script lang="ts" setup>
import { h, ref, onMounted } from 'vue';

import { Page, useVbenModal, type VbenFormProps } from 'shell/vben/common-ui';
import { LucideEye, LucideTrash, LucidePencil } from 'shell/vben/icons';

import { notification, Space, Button, Tag } from 'ant-design-vue';

import { useVbenVxeGrid } from 'shell/adapter/vxe-table';
import type { VxeGridProps } from 'shell/adapter/vxe-table';
import type { AllowancePool, AbsenceType } from '../../api/client';
import { $t } from 'shell/locales';
import { useHrAllowancePoolStore } from '../../stores/hr-allowance-pool.state';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';

import PoolDrawer from './pool-drawer.vue';
import { usePermission } from '../../composables/use-permission';

const poolStore = useHrAllowancePoolStore();
const absenceTypeStore = useHrAbsenceTypeStore();
const { canManagePools } = usePermission();

const typeMap = ref<Record<string, AbsenceType>>({});

onMounted(async () => {
  try {
    const resp = await absenceTypeStore.listAbsenceTypes(undefined, null);
    const items = (resp as { items: AbsenceType[] }).items || [];
    const map: Record<string, AbsenceType> = {};
    for (const t of items) {
      if (t.id) map[t.id] = t;
    }
    typeMap.value = map;
  } catch { /* silently fail */ }
});

function getTypeName(id: string): string {
  return typeMap.value[id]?.name || id;
}

function getTypeColor(id: string): string | undefined {
  return typeMap.value[id]?.color;
}

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

const gridOptions: VxeGridProps<AllowancePool> = {
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
        const resp = await poolStore.listAllowancePools(
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
      title: $t('hr.page.pool.name'),
      field: 'name',
      minWidth: 150,
      slots: { default: 'nameSlot' },
    },
    {
      title: $t('hr.page.pool.description'),
      field: 'description',
      minWidth: 200,
    },
    {
      title: $t('hr.page.pool.absenceTypes'),
      field: 'absenceTypeIds',
      minWidth: 200,
      slots: { default: 'typesSlot' },
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

const [PoolDrawerComponent, poolDrawerApi] = useVbenModal({
  connectedComponent: PoolDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.query();
    }
  },
});

function openModal(row: AllowancePool, mode: 'create' | 'edit' | 'view') {
  poolDrawerApi.setData({ row, mode });
  poolDrawerApi.open();
}

function handleView(row: AllowancePool) {
  openModal(row, 'view');
}

function handleEdit(row: AllowancePool) {
  openModal(row, 'edit');
}

function handleCreate() {
  openModal({} as AllowancePool, 'create');
}

async function handleDelete(row: AllowancePool) {
  if (!row.id) return;
  try {
    await poolStore.deleteAllowancePool(row.id);
    notification.success({
      message: $t('hr.page.pool.deleteSuccess'),
    });
    await gridApi.query();
  } catch (e: any) {
    notification.error({ message: $t('ui.notification.delete_failed'), description: e?.message });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('hr.page.pool.title')">
      <template #toolbar-tools>
        <Button v-if="canManagePools" class="mr-2" type="primary" @click="handleCreate">
          {{ $t('hr.page.pool.create') }}
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
      <template #typesSlot="{ row }">
        <span v-if="!row.absenceTypeIds?.length" class="text-gray-400">
          {{ $t('hr.page.pool.noTypes') }}
        </span>
        <Tag v-for="id in (row.absenceTypeIds || [])" :key="id" :color="getTypeColor(id)" class="mb-1">
          {{ getTypeName(id) }}
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
            v-if="canManagePools"
            type="link"
            size="small"
            :icon="h(LucidePencil)"
            :title="$t('ui.button.edit')"
            @click.stop="handleEdit(row)"
          />
          <a-popconfirm
            v-if="canManagePools"
            :cancel-text="$t('ui.button.cancel')"
            :ok-text="$t('ui.button.ok')"
            :title="$t('hr.page.pool.confirmDelete')"
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

    <PoolDrawerComponent />
  </Page>
</template>
