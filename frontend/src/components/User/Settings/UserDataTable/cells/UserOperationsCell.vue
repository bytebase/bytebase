<template>
  <div class="flex justify-end">
    <template v-if="allowEdit">
      <NPopconfirm v-if="allowDeleteUser" @positive-click="handleArchiveUser">
        <template #trigger>
          <NButton quaternary circle @click.stop>
            <template #icon>
              <Trash2Icon class="w-4 h-auto" />
            </template>
          </NButton>
        </template>

        <template #default>
          <div>
            {{ $t("settings.members.action.deactivate-confirm-title") }}
          </div>
        </template>
      </NPopconfirm>

      <NButton
        v-if="allowUpdateUser"
        quaternary
        circle
        @click="(e) => $emit('click-user', user, e)"
      >
        <template #icon>
          <PencilIcon class="w-4 h-auto" />
        </template>
      </NButton>

      <BBButtonConfirm
        v-if="allowReactiveUser"
        :type="'RESTORE'"
        :require-confirm="true"
        :ok-text="$t('settings.members.action.reactivate')"
        :confirm-title="`${$t(
          'settings.members.action.reactivate-confirm-title'
        )} '${user.title}'?`"
        :confirm-description="''"
        @confirm="changeRowStatus(State.ACTIVE)"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NButton, NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import { useUserStore, pushNotification, useCurrentUserV1 } from "@/store";
import { type User, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "click-user", user: User, e: MouseEvent): void;
}>();

const userStore = useUserStore();
const { t } = useI18n();
const me = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.users.update");
});

const allowUpdateUser = computed(() => {
  if (props.user.userType === UserType.SYSTEM_BOT) {
    return false;
  }
  return props.user.state === State.ACTIVE;
});

const allowDeleteUser = computed(() => {
  if (!allowUpdateUser.value) {
    return false;
  }
  // cannot delete self.
  return me.value.name !== props.user.name;
});

const handleArchiveUser = async () => {
  await userStore.archiveUser(props.user!);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.archived"),
  });
};

const allowReactiveUser = computed(() => {
  return allowEdit.value && props.user.state === State.DELETED;
});

const changeRowStatus = (state: State) => {
  if (state === State.ACTIVE) {
    userStore.restoreUser(props.user);
  } else {
    userStore.archiveUser(props.user);
  }
};
</script>
