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
  DatePicker,
  Descriptions,
  DescriptionsItem,
  Tag,
} from 'ant-design-vue';

import type { LeaveRequest, AbsenceType } from '../../api/services';
import { $t } from 'shell/locales';
import { useUserStore } from 'shell/vben/stores';
import { useHrLeaveStore } from '../../stores/hr-leave.state';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';
import { adminApi } from '../../api/client';

const leaveStore = useHrLeaveStore();
const absenceTypeStore = useHrAbsenceTypeStore();
const userStore = useUserStore();

interface PortalUser {
  id: number;
  username?: string;
  realname?: string;
  orgUnitNames?: string[];
}

const data = ref<{
  mode: 'create' | 'edit' | 'view';
  row?: LeaveRequest;
}>();
const loading = ref(false);
const absenceTypes = ref<AbsenceType[]>([]);
const users = ref<PortalUser[]>([]);

const formState = ref({
  userId: undefined as number | undefined,
  userName: '',
  orgUnitName: '',
  absenceTypeId: '',
  startDate: '',
  endDate: '',
  reason: '',
  notes: '',
});

const title = computed(() => {
  switch (data.value?.mode) {
    case 'create':
      return $t('hr.page.request.create');
    case 'edit':
      return $t('hr.page.request.edit');
    default:
      return $t('hr.page.request.view');
  }
});

const isCreateMode = computed(() => data.value?.mode === 'create');
const isEditMode = computed(() => data.value?.mode === 'edit');
const isViewMode = computed(() => data.value?.mode === 'view');

function statusColor(status?: string): string {
  switch (status) {
    case 'LEAVE_REQUEST_STATUS_APPROVED': return 'green';
    case 'LEAVE_REQUEST_STATUS_REJECTED': return 'red';
    case 'LEAVE_REQUEST_STATUS_CANCELLED': return 'default';
    case 'LEAVE_REQUEST_STATUS_PENDING': return 'orange';
    default: return 'default';
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case 'LEAVE_REQUEST_STATUS_APPROVED': return $t('hr.enum.leaveRequestStatus.approved');
    case 'LEAVE_REQUEST_STATUS_REJECTED': return $t('hr.enum.leaveRequestStatus.rejected');
    case 'LEAVE_REQUEST_STATUS_CANCELLED': return $t('hr.enum.leaveRequestStatus.cancelled');
    case 'LEAVE_REQUEST_STATUS_PENDING': return $t('hr.enum.leaveRequestStatus.pending');
    default: return '';
  }
}

function getUserDisplayName(user: PortalUser): string {
  return user.realname || user.username || '';
}

function onUserSelect(userId: number) {
  const user = users.value.find((u) => u.id === userId);
  if (user) {
    formState.value.userName = getUserDisplayName(user);
    formState.value.orgUnitName = user.orgUnitNames?.[0] || '';
  }
}

async function loadOptions() {
  try {
    const [typesResp, usersResp] = await Promise.all([
      absenceTypeStore.listAbsenceTypes(undefined, null),
      adminApi.get<{ items: PortalUser[] }>('/users', { noPaging: true }),
    ]);
    absenceTypes.value = typesResp.items || [];
    users.value = usersResp.items || [];
  } catch {
    // silently fail
  }
}

async function handleSubmit() {
  loading.value = true;
  try {
    if (isCreateMode.value) {
      await leaveStore.createLeaveRequest({
        userId: formState.value.userId,
        userName: formState.value.userName || undefined,
        orgUnitName: formState.value.orgUnitName || undefined,
        absenceTypeId: formState.value.absenceTypeId,
        startDate: formState.value.startDate,
        endDate: formState.value.endDate,
        reason: formState.value.reason || undefined,
        notes: formState.value.notes || undefined,
      });
      notification.success({
        message: $t('hr.page.request.createSuccess'),
      });
    } else if (isEditMode.value && data.value?.row?.id) {
      await leaveStore.updateLeaveRequest(
        data.value.row.id,
        {
          reason: formState.value.reason || undefined,
          notes: formState.value.notes || undefined,
        },
        ['reason', 'notes'],
      );
      notification.success({
        message: $t('hr.page.request.updateSuccess'),
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
    userName: '',
    orgUnitName: '',
    absenceTypeId: '',
    startDate: '',
    endDate: '',
    reason: '',
    notes: '',
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
        row?: LeaveRequest;
      };

      await loadOptions();

      if (data.value?.mode === 'create') {
        resetForm();
        // Pre-fill from drag-to-create or other sources
        if (data.value?.row) {
          if (data.value.row.userId) {
            formState.value.userId = data.value.row.userId;
            onUserSelect(data.value.row.userId);
          }
          if (data.value.row.absenceTypeId) formState.value.absenceTypeId = data.value.row.absenceTypeId;
          if (data.value.row.startDate) formState.value.startDate = data.value.row.startDate;
          if (data.value.row.endDate) formState.value.endDate = data.value.row.endDate;
        }
        // Default to current logged-in user if not pre-filled
        if (!formState.value.userId && userStore.userInfo) {
          formState.value.userId = userStore.userInfo.id;
          formState.value.userName = userStore.userInfo.realname || userStore.userInfo.username || '';
        }
      } else if (data.value?.row) {
        formState.value = {
          userId: data.value.row.userId,
          userName: data.value.row.userName ?? '',
          orgUnitName: data.value.row.orgUnitName ?? '',
          absenceTypeId: data.value.row.absenceTypeId ?? '',
          startDate: data.value.row.startDate ?? '',
          endDate: data.value.row.endDate ?? '',
          reason: data.value.row.reason ?? '',
          notes: data.value.row.notes ?? '',
        };
      }
    }
  },
});

const request = computed(() => data.value?.row);
</script>

<template>
  <Modal :title="title" :footer="false" class="w-[600px]">
    <!-- View Mode -->
    <template v-if="request && isViewMode">
      <Descriptions :column="1" bordered size="small">
        <DescriptionsItem :label="$t('hr.page.request.userName')">
          {{ request.userName || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.request.absenceTypeName')">
          {{ request.absenceTypeName || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.request.startDate')">
          {{ request.startDate || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.request.endDate')">
          {{ request.endDate || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.request.days')">
          {{ request.days || '-' }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('hr.page.request.status')">
          <Tag :color="statusColor(request.status)">
            {{ statusLabel(request.status) }}
          </Tag>
        </DescriptionsItem>
        <DescriptionsItem v-if="request.reason" :label="$t('hr.page.request.reason')">
          {{ request.reason }}
        </DescriptionsItem>
        <DescriptionsItem v-if="request.reviewNotes" :label="$t('hr.page.request.reviewNotes')">
          {{ request.reviewNotes }}
        </DescriptionsItem>
        <DescriptionsItem v-if="request.notes" :label="$t('hr.page.request.notes')">
          {{ request.notes }}
        </DescriptionsItem>
      </Descriptions>
    </template>

    <!-- Create/Edit Mode -->
    <template v-else-if="isCreateMode || isEditMode">
      <Form layout="vertical" :model="formState" @finish="handleSubmit">
        <FormItem
          v-if="isCreateMode"
          :label="$t('hr.page.request.userId')"
          name="userId"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <Select
            v-model:value="formState.userId"
            :placeholder="$t('ui.placeholder.select')"
            show-search
            option-filter-prop="label"
            @change="onUserSelect"
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
          v-if="isCreateMode"
          :label="$t('hr.page.request.absenceTypeId')"
          name="absenceTypeId"
          :rules="[{ required: true, message: $t('ui.formRules.required') }]"
        >
          <Select
            v-model:value="formState.absenceTypeId"
            :placeholder="$t('ui.placeholder.select')"
          >
            <SelectOption
              v-for="at in absenceTypes"
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

        <div v-if="isCreateMode" class="flex gap-4">
          <FormItem
            class="flex-1"
            :label="$t('hr.page.request.startDate')"
            name="startDate"
            :rules="[{ required: true, message: $t('ui.formRules.required') }]"
          >
            <Input
              v-model:value="formState.startDate"
              type="date"
              :placeholder="$t('ui.placeholder.input')"
            />
          </FormItem>

          <FormItem
            class="flex-1"
            :label="$t('hr.page.request.endDate')"
            name="endDate"
            :rules="[{ required: true, message: $t('ui.formRules.required') }]"
          >
            <Input
              v-model:value="formState.endDate"
              type="date"
              :placeholder="$t('ui.placeholder.input')"
            />
          </FormItem>
        </div>

        <FormItem :label="$t('hr.page.request.reason')" name="reason">
          <Textarea
            v-model:value="formState.reason"
            :rows="3"
            :maxlength="1024"
            :placeholder="$t('ui.placeholder.input')"
          />
        </FormItem>

        <FormItem :label="$t('hr.page.request.notes')" name="notes">
          <Textarea
            v-model:value="formState.notes"
            :rows="2"
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
