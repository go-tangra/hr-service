<script lang="ts" setup>
import { ref, computed } from 'vue';

import { useVbenModal } from 'shell/vben/common-ui';

import {
  Form,
  FormItem,
  InputNumber,
  Button,
  notification,
  Textarea,
  Select,
  SelectOption,
  Descriptions,
  DescriptionsItem,
  RadioGroup,
  RadioButton,
} from 'ant-design-vue';

import { userService, type LeaveAllowance, type AbsenceType, type AllowancePool, type HrUser } from '../../api/client';
import { $t } from 'shell/locales';
import { useHrAllowanceStore } from '../../stores/hr-allowance.state';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';
import { useHrAllowancePoolStore } from '../../stores/hr-allowance-pool.state';

const allowanceStore = useHrAllowanceStore();
const absenceTypeStore = useHrAbsenceTypeStore();
const poolStore = useHrAllowancePoolStore();

const data = ref<{
  mode: 'create' | 'edit' | 'view';
  row?: LeaveAllowance;
}>();
const loading = ref(false);
const absenceTypes = ref<AbsenceType[]>([]);
const pools = ref<AllowancePool[]>([]);
const users = ref<HrUser[]>([]);

const formState = ref({
  userId: undefined as number | undefined,
  assignmentType: 'type' as 'type' | 'pool',
  absenceTypeId: '',
  allowancePoolId: '',
  year: new Date().getFullYear(),
  totalDays: 0,
  carriedOver: 0,
  notes: '',
});

const title = computed(() => {
  switch (data.value?.mode) {
    case 'create':
      return $t('hr.page.allowance.create');
    case 'edit':
      return $t('hr.page.allowance.edit');
    default:
      return $t('hr.page.allowance.view');
  }
});

const isCreateMode = computed(() => data.value?.mode === 'create');
const isEditMode = computed(() => data.value?.mode === 'edit');
const isViewMode = computed(() => data.value?.mode === 'view');

// Filter to only absence types that deduct from allowance
const deductingTypes = computed(() =>
  absenceTypes.value.filter((t) => t.deductsFromAllowance),
);

function getUserDisplayName(user: HrUser): string {
  return user.realname || user.username || '';
}

async function loadOptions() {
  try {
    const [typesResp, usersResp, poolsResp] = await Promise.all([
      absenceTypeStore.listAbsenceTypes(undefined, null),
      userService.ListUsers({ noPaging: true }),
      poolStore.listAllowancePools(undefined, null),
    ]);
    absenceTypes.value = (typesResp as { items: AbsenceType[] }).items || [];
    users.value = usersResp.items || [];
    pools.value = poolsResp.items || [];
  } catch {
    // silently fail
  }
}

async function handleSubmit() {
  loading.value = true;
  try {
    if (isCreateMode.value) {
      const selectedUser = users.value.find(
        (u) => u.id === formState.value.userId,
      );
      const createData: any = {
        userId: formState.value.userId,
        userName: selectedUser ? getUserDisplayName(selectedUser) : undefined,
        year: formState.value.year,
        totalDays: formState.value.totalDays,
        carriedOver: formState.value.carriedOver,
        notes: formState.value.notes || undefined,
      };
      if (formState.value.assignmentType === 'pool') {
        createData.allowancePoolId = formState.value.allowancePoolId;
      } else {
        createData.absenceTypeId = formState.value.absenceTypeId;
      }
      await allowanceStore.createAllowance(createData);
      notification.success({
        message: $t('hr.page.allowance.createSuccess'),
      });
    } else if (isEditMode.value && data.value?.row?.id) {
      await allowanceStore.updateAllowance(
        data.value.row.id,
        {
          totalDays: formState.value.totalDays,
          carriedOver: formState.value.carriedOver,
          notes: formState.value.notes || undefined,
        },
        ['totalDays', 'carriedOver', 'notes'],
      );
      notification.success({
        message: $t('hr.page.allowance.updateSuccess'),
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
    userId: undefined,
    assignmentType: 'type',
    absenceTypeId: '',
    allowancePoolId: '',
    year: new Date().getFullYear(),
    totalDays: 0,
    carriedOver: 0,
    notes: '',
  };
}

function computeRemaining(row?: LeaveAllowance): number {
  if (!row) return 0;
  return (row.totalDays ?? 0) + (row.carriedOver ?? 0) - (row.usedDays ?? 0);
}

const [Modal, modalApi] = useVbenModal({
  onCancel() {
    modalApi.close();
  },

  async onOpenChange(isOpen) {
    if (isOpen) {
      data.value = modalApi.getData() as {
        mode: 'create' | 'edit' | 'view';
        row?: LeaveAllowance;
      };

      await loadOptions();

      if (data.value?.mode === 'create') {
        resetForm();
      } else if (data.value?.row) {
        formState.value = {
          userId: data.value.row.userId,
          assignmentType: data.value.row.allowancePoolId ? 'pool' : 'type',
          absenceTypeId: data.value.row.absenceTypeId ?? '',
          allowancePoolId: data.value.row.allowancePoolId ?? '',
          year: data.value.row.year ?? new Date().getFullYear(),
          totalDays: data.value.row.totalDays ?? 0,
          carriedOver: data.value.row.carriedOver ?? 0,
          notes: data.value.row.notes ?? '',
        };
      }
    }
  },
});

const allowance = computed(() => data.value?.row);
</script>

<template>
  <Modal :title="title" :footer="false" class="w-[600px]">
    <!-- View Mode -->
    <template v-if="allowance && isViewMode">
      <Descriptions :column="1" bordered size="small">
        <DescriptionsItem :label="$t('hr.page.allowance.userId')">
          {{ allowance.userName || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.absenceTypeId')">
          {{ allowance.absenceTypeName || '-' }}
        </DescriptionsItem>
        <DescriptionsItem v-if="allowance.allowancePoolId" :label="$t('hr.page.allowance.poolName')">
          {{ allowance.allowancePoolName || allowance.allowancePoolId }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.year')">
          {{ allowance.year ?? '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.totalDays')">
          {{ allowance.totalDays ?? 0 }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.usedDays')">
          {{ allowance.usedDays ?? 0 }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.carriedOver')">
          {{ allowance.carriedOver ?? 0 }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.allowance.remaining')">
          <span
            :style="{
              color: computeRemaining(allowance) <= 0 ? '#f5222d' : '#52c41a',
              fontWeight: 600,
            }"
          >
            {{ computeRemaining(allowance) }}
          </span>
        </DescriptionsItem>
        <DescriptionsItem v-if="allowance.notes" :label="$t('hr.page.allowance.notes')">
          {{ allowance.notes }}
        </DescriptionsItem>
      </Descriptions>
    </template>

    <!-- Create/Edit Mode -->
    <template v-else-if="isCreateMode || isEditMode">
      <Form layout="vertical" :model="formState" @finish="handleSubmit">
        <FormItem
          v-if="isCreateMode"
          :label="$t('hr.page.allowance.userId')"
          name="userId"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <Select
            v-model:value="formState.userId"
            :placeholder="$t('ui.placeholder.select')"
            show-search
            option-filter-prop="label"
          >
            <SelectOption
              v-for="user in users"
              :key="user.id"
              :value="user.id"
              :label="getUserDisplayName(user)"
            >
              {{ getUserDisplayName(user) }}
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem
          v-if="isCreateMode && pools.length > 0"
          :label="$t('hr.page.allowance.assignmentType')"
          name="assignmentType"
        >
          <RadioGroup v-model:value="formState.assignmentType" button-style="solid" size="small">
            <RadioButton value="type">{{ $t('hr.page.allowance.assignAbsenceType') }}</RadioButton>
            <RadioButton value="pool">{{ $t('hr.page.allowance.assignPool') }}</RadioButton>
          </RadioGroup>
        </FormItem>

        <FormItem
          v-if="isCreateMode && formState.assignmentType === 'type'"
          :label="$t('hr.page.allowance.absenceTypeId')"
          name="absenceTypeId"
          :rules="[{ required: formState.assignmentType === 'type', message: $t('ui.formRules.required') }]"
        >
          <Select
            v-model:value="formState.absenceTypeId"
            :placeholder="$t('ui.placeholder.select')"
          >
            <SelectOption
              v-for="at in deductingTypes"
              :key="at.id"
              :value="at.id"
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

        <FormItem
          v-if="isCreateMode && formState.assignmentType === 'pool'"
          :label="$t('hr.page.allowance.poolName')"
          name="allowancePoolId"
          :rules="[{ required: formState.assignmentType === 'pool', message: $t('ui.formRules.required') }]"
        >
          <Select
            v-model:value="formState.allowancePoolId"
            :placeholder="$t('ui.placeholder.select')"
          >
            <SelectOption
              v-for="pool in pools"
              :key="pool.id"
              :value="pool.id"
            >
              <span
                v-if="pool.color"
                class="mr-1 inline-block h-3 w-3 rounded-full"
                :style="{ backgroundColor: pool.color }"
              />
              {{ pool.name }}
            </SelectOption>
          </Select>
        </FormItem>

        <FormItem
          v-if="isCreateMode"
          :label="$t('hr.page.allowance.year')"
          name="year"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <InputNumber
            v-model:value="formState.year"
            :min="2000"
            :max="2099"
            style="width: 120px"
          />
        </FormItem>

        <div class="flex gap-4">
          <FormItem
            class="flex-1"
            :label="$t('hr.page.allowance.totalDays')"
            name="totalDays"
            :rules="[{ required: true, message: $t('ui.formRules.required') }]"
          >
            <InputNumber
              v-model:value="formState.totalDays"
              :min="0"
              :max="365"
              :step="0.5"
              style="width: 100%"
            />
          </FormItem>

          <FormItem
            class="flex-1"
            :label="$t('hr.page.allowance.carriedOver')"
            name="carriedOver"
          >
            <InputNumber
              v-model:value="formState.carriedOver"
              :min="0"
              :max="365"
              :step="0.5"
              style="width: 100%"
            />
          </FormItem>
        </div>

        <FormItem :label="$t('hr.page.allowance.notes')" name="notes">
          <Textarea
            v-model:value="formState.notes"
            :rows="3"
            :maxlength="1024"
            :placeholder="$t('ui.placeholder.input')"
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
