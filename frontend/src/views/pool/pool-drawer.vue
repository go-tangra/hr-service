<script lang="ts" setup>
import { ref, computed } from 'vue';

import { useVbenModal } from 'shell/vben/common-ui';

import {
  Form,
  FormItem,
  Input,
  Button,
  notification,
  Textarea,
  Select,
  SelectOption,
  Descriptions,
  DescriptionsItem,
  Tag,
} from 'ant-design-vue';

import type { AllowancePool, AbsenceType } from '../../api/client';
import { $t } from 'shell/locales';
import { useHrAllowancePoolStore } from '../../stores/hr-allowance-pool.state';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';

const poolStore = useHrAllowancePoolStore();
const absenceTypeStore = useHrAbsenceTypeStore();

const data = ref<{
  mode: 'create' | 'edit' | 'view';
  row?: AllowancePool;
}>();
const loading = ref(false);
const absenceTypes = ref<AbsenceType[]>([]);

const formState = ref({
  name: '',
  description: '',
  color: '#1890ff',
  icon: '',
  absenceTypeIds: [] as string[],
});

const title = computed(() => {
  switch (data.value?.mode) {
    case 'create':
      return $t('hr.page.pool.create');
    case 'edit':
      return $t('hr.page.pool.edit');
    default:
      return $t('hr.page.pool.view');
  }
});

const isCreateMode = computed(() => data.value?.mode === 'create');
const isEditMode = computed(() => data.value?.mode === 'edit');
const isViewMode = computed(() => data.value?.mode === 'view');

// Only show absence types that deduct from allowance
const deductingTypes = computed(() =>
  absenceTypes.value.filter((t) => t.deductsFromAllowance),
);

function getTypeName(id: string): string {
  return absenceTypes.value.find((t) => t.id === id)?.name || id;
}

async function handleSubmit() {
  loading.value = true;
  try {
    if (isCreateMode.value) {
      await poolStore.createAllowancePool({
        name: formState.value.name,
        description: formState.value.description || undefined,
        color: formState.value.color || undefined,
        icon: formState.value.icon || undefined,
        absenceTypeIds: formState.value.absenceTypeIds,
      });
      notification.success({
        message: $t('hr.page.pool.createSuccess'),
      });
    } else if (isEditMode.value && data.value?.row?.id) {
      await poolStore.updateAllowancePool(
        data.value.row.id,
        {
          name: formState.value.name,
          description: formState.value.description || undefined,
          color: formState.value.color || undefined,
          icon: formState.value.icon || undefined,
        },
        ['name', 'description', 'color', 'icon'],
      );
      notification.success({
        message: $t('hr.page.pool.updateSuccess'),
      });
    }
    modalApi.close();
  } catch {
    notification.error({
      message: isCreateMode.value
        ? $t('ui.notification.create_failed')
        : $t('ui.notification.update_failed'),
    });
  } finally {
    loading.value = false;
  }
}

function resetForm() {
  formState.value = {
    name: '',
    description: '',
    color: '#1890ff',
    icon: '',
    absenceTypeIds: [],
  };
}

const [Modal, modalApi] = useVbenModal({
  onCancel() {
    modalApi.close();
  },

  async onOpenChange(isOpen) {
    if (isOpen) {
      data.value = modalApi.getData() as {
        mode: 'create' | 'edit' | 'view';
        row?: AllowancePool;
      };

      // Load absence types for the multi-select
      try {
        const resp = await absenceTypeStore.listAbsenceTypes(undefined, null);
        absenceTypes.value = (resp as { items: AbsenceType[] }).items || [];
      } catch {
        absenceTypes.value = [];
      }

      if (data.value?.mode === 'create') {
        resetForm();
      } else if (data.value?.row) {
        formState.value = {
          name: data.value.row.name ?? '',
          description: data.value.row.description ?? '',
          color: data.value.row.color ?? '#1890ff',
          icon: data.value.row.icon ?? '',
          absenceTypeIds: data.value.row.absenceTypeIds ?? [],
        };
      }
    }
  },
});

const pool = computed(() => data.value?.row);
</script>

<template>
  <Modal :title="title" :footer="false" class="w-[600px]">
    <!-- View Mode -->
    <template v-if="pool && isViewMode">
      <Descriptions :column="1" bordered size="small">
        <DescriptionsItem :label="$t('hr.page.pool.name')">
          <span class="flex items-center gap-2">
            <span
              v-if="pool.color"
              class="inline-block h-3 w-3 rounded-full"
              :style="{ backgroundColor: pool.color }"
            />
            {{ pool.name }}
          </span>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.pool.description')">
          {{ pool.description || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.pool.color')">
          <span class="flex items-center gap-2">
            <span
              class="inline-block h-4 w-4 rounded border"
              :style="{ backgroundColor: pool.color }"
            />
            {{ pool.color || '-' }}
          </span>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.pool.icon')">
          {{ pool.icon || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.pool.absenceTypes')">
          <template v-if="pool.absenceTypeIds && pool.absenceTypeIds.length">
            <Tag v-for="id in pool.absenceTypeIds" :key="id" class="mb-1">
              {{ getTypeName(id) }}
            </Tag>
          </template>
          <span v-else class="text-gray-400">{{ $t('hr.page.pool.noTypes') }}</span>
        </DescriptionsItem>
      </Descriptions>
    </template>

    <!-- Create/Edit Mode -->
    <template v-else-if="isCreateMode || isEditMode">
      <Form layout="vertical" :model="formState" @finish="handleSubmit">
        <FormItem
          :label="$t('hr.page.pool.name')"
          name="name"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <Input
            v-model:value="formState.name"
            :placeholder="$t('ui.placeholder.input')"
            :maxlength="255"
          />
        </FormItem>

        <FormItem :label="$t('hr.page.pool.description')" name="description">
          <Textarea
            v-model:value="formState.description"
            :rows="3"
            :maxlength="1024"
            :placeholder="$t('ui.placeholder.input')"
          />
        </FormItem>

        <div class="flex gap-4">
          <FormItem class="flex-1" :label="$t('hr.page.pool.color')" name="color">
            <Input
              v-model:value="formState.color"
              type="color"
              style="width: 60px; height: 32px; padding: 2px"
            />
          </FormItem>

          <FormItem class="flex-1" :label="$t('hr.page.pool.icon')" name="icon">
            <Input
              v-model:value="formState.icon"
              :placeholder="$t('ui.placeholder.input')"
              :maxlength="100"
            />
          </FormItem>
        </div>

        <FormItem
          v-if="isCreateMode"
          :label="$t('hr.page.pool.absenceTypes')"
          name="absenceTypeIds"
        >
          <Select
            v-model:value="formState.absenceTypeIds"
            mode="multiple"
            :placeholder="$t('ui.placeholder.select')"
            option-filter-prop="label"
          >
            <SelectOption
              v-for="at in deductingTypes"
              :key="at.id"
              :value="at.id"
              :label="at.name"
            >
              <span
                v-if="at.color"
                class="mr-1 inline-block h-3 w-3 rounded-full"
                :style="{ backgroundColor: at.color }"
              />
              {{ at.name }}
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem class="mt-4">
          <Button type="primary" html-type="submit" :loading="loading" block>
            {{
              isCreateMode
                ? $t('ui.button.create', { moduleName: '' })
                : $t('ui.button.save')
            }}
          </Button>
        </FormItem>
      </Form>
    </template>
  </Modal>
</template>
