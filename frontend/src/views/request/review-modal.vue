<script lang="ts" setup>
import { ref, computed } from 'vue';

import { useVbenModal } from 'shell/vben/common-ui';

import {
  Form,
  FormItem,
  Button,
  notification,
  Textarea,
} from 'ant-design-vue';

import { $t } from 'shell/locales';
import { useUserStore } from 'shell/vben/stores';
import { useHrLeaveStore } from '../../stores/hr-leave.state';
import { adminApi } from '../../api/client';
import type { LeaveRequest } from '../../api/services';

interface PortalUser {
  id: number;
  username?: string;
  realname?: string;
  email?: string;
}

const leaveStore = useHrLeaveStore();
const userStore = useUserStore();

const data = ref<{
  action: 'approve' | 'reject';
  requestId: string;
  row?: LeaveRequest;
}>();
const loading = ref(false);
const requesterEmail = ref('');

const formState = ref({
  reviewNotes: '',
});

const title = computed(() => {
  return data.value?.action === 'approve'
    ? $t('hr.page.request.approve')
    : $t('hr.page.request.reject');
});

const isApprove = computed(() => data.value?.action === 'approve');

async function fetchRequesterEmail(userId: number) {
  try {
    const resp = await adminApi.get<{ user: PortalUser }>(`/users/${userId}`);
    requesterEmail.value = resp.user?.email || '';
  } catch {
    requesterEmail.value = '';
  }
}

async function handleSubmit() {
  if (!data.value?.requestId) return;
  loading.value = true;
  try {
    if (isApprove.value) {
      const currentUser = userStore.userInfo;

      await leaveStore.approveLeaveRequest(
        data.value.requestId,
        formState.value.reviewNotes || undefined,
        currentUser?.email || undefined,
        currentUser?.realname || currentUser?.username || undefined,
        requesterEmail.value || undefined,
      );
      notification.success({
        message: $t('hr.page.request.approveSuccess'),
      });
    } else {
      await leaveStore.rejectLeaveRequest(
        data.value.requestId,
        formState.value.reviewNotes || undefined,
      );
      notification.success({
        message: $t('hr.page.request.rejectSuccess'),
      });
    }
    modalApi.close();
  } catch {
    notification.error({ message: $t('ui.notification.update_failed') });
  } finally {
    loading.value = false;
  }
}

const [Modal, modalApi] = useVbenModal({
  onCancel() {
    modalApi.close();
  },

  async onOpenChange(isOpen) {
    if (isOpen) {
      data.value = modalApi.getData() as {
        action: 'approve' | 'reject';
        requestId: string;
        row?: LeaveRequest;
      };
      formState.value = { reviewNotes: '' };
      requesterEmail.value = '';

      // Fetch requester email for signing when approving
      if (data.value?.action === 'approve' && data.value?.row?.userId) {
        fetchRequesterEmail(data.value.row.userId);
      }
    }
  },
});
</script>

<template>
  <Modal :title="title" :footer="false" class="w-[450px]">
    <Form layout="vertical" :model="formState" @finish="handleSubmit">
      <FormItem :label="$t('hr.page.request.reviewNotes')" name="reviewNotes">
        <Textarea
          v-model:value="formState.reviewNotes"
          :rows="4"
          :maxlength="1024"
          :placeholder="$t('ui.placeholder.input')"
        />
      </FormItem>

      <FormItem class="mt-4">
        <Button
          :type="isApprove ? 'primary' : 'default'"
          :danger="!isApprove"
          html-type="submit"
          :loading="loading"
          block
        >
          {{ isApprove ? $t('hr.page.request.approve') : $t('hr.page.request.reject') }}
        </Button>
      </FormItem>
    </Form>
  </Modal>
</template>
