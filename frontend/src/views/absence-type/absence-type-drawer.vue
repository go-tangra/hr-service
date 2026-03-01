<script lang="ts" setup>
import { ref, computed } from 'vue';

import { useVbenModal } from 'shell/vben/common-ui';

import {
  Form,
  FormItem,
  Input,
  InputNumber,
  Button,
  notification,
  Textarea,
  Switch,
  Descriptions,
  DescriptionsItem,
  Tag,
  Select,
  SelectOption,
} from 'ant-design-vue';

import type { AbsenceType } from '../../api/services';
import { paperlessApi } from '../../api/client';
import { $t } from 'shell/locales';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';

const absenceTypeStore = useHrAbsenceTypeStore();

interface SigningTemplate {
  id: string;
  name: string;
}

const data = ref<{
  mode: 'create' | 'edit' | 'view';
  row?: AbsenceType;
}>();
const loading = ref(false);
const signingTemplates = ref<SigningTemplate[]>([]);

const formState = ref({
  name: '',
  description: '',
  color: '#1890ff',
  icon: '',
  deductsFromAllowance: true,
  requiresApproval: true,
  isActive: true,
  sortOrder: 0,
  metadata: '',
  requiresSigning: false,
  signingTemplateId: '',
});

const title = computed(() => {
  switch (data.value?.mode) {
    case 'create':
      return $t('hr.page.absenceType.create');
    case 'edit':
      return $t('hr.page.absenceType.edit');
    default:
      return $t('hr.page.absenceType.view');
  }
});

const isCreateMode = computed(() => data.value?.mode === 'create');
const isEditMode = computed(() => data.value?.mode === 'edit');
const isViewMode = computed(() => data.value?.mode === 'view');

async function handleSubmit() {
  loading.value = true;
  try {
    if (isCreateMode.value) {
      await absenceTypeStore.createAbsenceType({
        name: formState.value.name,
        description: formState.value.description || undefined,
        color: formState.value.color || undefined,
        icon: formState.value.icon || undefined,
        deductsFromAllowance: formState.value.deductsFromAllowance,
        requiresApproval: formState.value.requiresApproval,
        isActive: formState.value.isActive,
        sortOrder: formState.value.sortOrder,
        metadata: formState.value.metadata ? JSON.parse(formState.value.metadata) : undefined,
        requiresSigning: formState.value.requiresSigning,
        signingTemplateId: formState.value.requiresSigning ? formState.value.signingTemplateId || undefined : undefined,
      });
      notification.success({
        message: $t('hr.page.absenceType.createSuccess'),
      });
    } else if (isEditMode.value && data.value?.row?.id) {
      await absenceTypeStore.updateAbsenceType(
        data.value.row.id,
        {
          name: formState.value.name,
          description: formState.value.description || undefined,
          color: formState.value.color || undefined,
          icon: formState.value.icon || undefined,
          deductsFromAllowance: formState.value.deductsFromAllowance,
          requiresApproval: formState.value.requiresApproval,
          isActive: formState.value.isActive,
          sortOrder: formState.value.sortOrder,
          metadata: formState.value.metadata ? JSON.parse(formState.value.metadata) : undefined,
          requiresSigning: formState.value.requiresSigning,
          signingTemplateId: formState.value.requiresSigning ? formState.value.signingTemplateId || undefined : undefined,
        },
        [
          'name',
          'description',
          'color',
          'icon',
          'deductsFromAllowance',
          'requiresApproval',
          'isActive',
          'sortOrder',
          'metadata',
          'requiresSigning',
          'signingTemplateId',
        ],
      );
      notification.success({
        message: $t('hr.page.absenceType.updateSuccess'),
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
    deductsFromAllowance: true,
    requiresApproval: true,
    isActive: true,
    sortOrder: 0,
    metadata: '',
    requiresSigning: false,
    signingTemplateId: '',
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
        row?: AbsenceType;
      };

      // Fetch signing templates from paperless module
      try {
        const resp = await paperlessApi.get<{ items: SigningTemplate[] }>('/signing/templates', { noPaging: true });
        signingTemplates.value = resp.items ?? [];
      } catch {
        signingTemplates.value = [];
      }

      if (data.value?.mode === 'create') {
        resetForm();
      } else if (data.value?.row) {
        formState.value = {
          name: data.value.row.name ?? '',
          description: data.value.row.description ?? '',
          color: data.value.row.color ?? '#1890ff',
          icon: data.value.row.icon ?? '',
          deductsFromAllowance: data.value.row.deductsFromAllowance ?? true,
          requiresApproval: data.value.row.requiresApproval ?? true,
          isActive: data.value.row.isActive ?? true,
          sortOrder: data.value.row.sortOrder ?? 0,
          metadata: data.value.row.metadata ? JSON.stringify(data.value.row.metadata, null, 2) : '',
          requiresSigning: data.value.row.requiresSigning ?? false,
          signingTemplateId: data.value.row.signingTemplateId ?? '',
        };
      }
    }
  },
});

const absenceType = computed(() => data.value?.row);
</script>

<template>
  <Modal :title="title" :footer="false" class="w-[600px]">
    <!-- View Mode -->
    <template v-if="absenceType && isViewMode">
      <Descriptions :column="1" bordered size="small">
        <DescriptionsItem :label="$t('hr.page.absenceType.name')">
          <span class="flex items-center gap-2">
            <span
              v-if="absenceType.color"
              class="inline-block h-3 w-3 rounded-full"
              :style="{ backgroundColor: absenceType.color }"
            />
            {{ absenceType.name }}
          </span>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.description')">
          {{ absenceType.description || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.color')">
          <span class="flex items-center gap-2">
            <span
              class="inline-block h-4 w-4 rounded border"
              :style="{ backgroundColor: absenceType.color }"
            />
            {{ absenceType.color || '-' }}
          </span>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.icon')">
          {{ absenceType.icon || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.deductsFromAllowance')">
          <Tag :color="absenceType.deductsFromAllowance ? 'blue' : 'default'">
            {{ absenceType.deductsFromAllowance ? 'Yes' : 'No' }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.requiresApproval')">
          <Tag :color="absenceType.requiresApproval ? 'orange' : 'default'">
            {{ absenceType.requiresApproval ? 'Yes' : 'No' }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.isActive')">
          <Tag :color="absenceType.isActive ? 'green' : 'red'">
            {{ absenceType.isActive ? 'Yes' : 'No' }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.requiresSigning')">
          <Tag :color="absenceType.requiresSigning ? 'purple' : 'default'">
            {{ absenceType.requiresSigning ? 'Yes' : 'No' }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem v-if="absenceType.requiresSigning" :label="$t('hr.page.absenceType.signingTemplate')">
          {{ signingTemplates.find(t => t.id === absenceType.signingTemplateId)?.name || absenceType.signingTemplateId || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.absenceType.sortOrder')">
          {{ absenceType.sortOrder ?? 0 }}
        </DescriptionsItem>
      </Descriptions>
    </template>

    <!-- Create/Edit Mode -->
    <template v-else-if="isCreateMode || isEditMode">
      <Form layout="vertical" :model="formState" @finish="handleSubmit">
        <FormItem
          :label="$t('hr.page.absenceType.name')"
          name="name"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <Input
            v-model:value="formState.name"
            :placeholder="$t('ui.placeholder.input')"
            :maxlength="255"
          />
        </FormItem>

        <FormItem :label="$t('hr.page.absenceType.description')" name="description">
          <Textarea
            v-model:value="formState.description"
            :rows="3"
            :maxlength="1024"
            :placeholder="$t('ui.placeholder.input')"
          />
        </FormItem>

        <div class="flex gap-4">
          <FormItem class="flex-1" :label="$t('hr.page.absenceType.color')" name="color">
            <Input
              v-model:value="formState.color"
              type="color"
              style="width: 60px; height: 32px; padding: 2px"
            />
          </FormItem>

          <FormItem class="flex-1" :label="$t('hr.page.absenceType.icon')" name="icon">
            <Input
              v-model:value="formState.icon"
              :placeholder="$t('ui.placeholder.input')"
              :maxlength="100"
            />
          </FormItem>
        </div>

        <div class="flex gap-8">
          <FormItem :label="$t('hr.page.absenceType.deductsFromAllowance')" name="deductsFromAllowance">
            <Switch v-model:checked="formState.deductsFromAllowance" />
          </FormItem>

          <FormItem :label="$t('hr.page.absenceType.requiresApproval')" name="requiresApproval">
            <Switch v-model:checked="formState.requiresApproval" />
          </FormItem>

          <FormItem :label="$t('hr.page.absenceType.isActive')" name="isActive">
            <Switch v-model:checked="formState.isActive" />
          </FormItem>
        </div>

        <div class="flex gap-8">
          <FormItem :label="$t('hr.page.absenceType.requiresSigning')" name="requiresSigning">
            <Switch v-model:checked="formState.requiresSigning" />
          </FormItem>
        </div>

        <FormItem
          v-if="formState.requiresSigning"
          :label="$t('hr.page.absenceType.signingTemplate')"
          name="signingTemplateId"
        >
          <Select
            v-model:value="formState.signingTemplateId"
            :placeholder="$t('ui.placeholder.input')"
            allow-clear
          >
            <SelectOption
              v-for="tpl in signingTemplates"
              :key="tpl.id"
              :value="tpl.id"
            >
              {{ tpl.name }}
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem :label="$t('hr.page.absenceType.sortOrder')" name="sortOrder">
          <InputNumber v-model:value="formState.sortOrder" :min="0" :max="9999" />
        </FormItem>

        <FormItem :label="$t('hr.page.absenceType.metadata')" name="metadata">
          <Textarea
            v-model:value="formState.metadata"
            :rows="4"
            placeholder='{"key": "value"}'
            style="font-family: monospace; font-size: 12px"
          />
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
