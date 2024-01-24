<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-96 max-w-full"
      :title="
        isCreating
          ? $t('settings.members.add-member')
          : $t('settings.members.update-member')
      "
    >
      <template #default>
        <NForm :model="state.user">
          <NFormItem path="email" :label="$t('common.email')">
            <div
              v-if="
                isCreating && state.user.userType === UserType.SERVICE_ACCOUNT
              "
              class="w-full flex flex-col items-start"
            >
              <NInput
                v-model:value="state.user.email"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                placeholder="foo"
              />
              <span class="mt-1 textinfolabel">
                {{ serviceAccountEmailSuffix }}
              </span>
            </div>
            <NInput
              v-else
              v-model:value="state.user.email"
              :input-props="{ type: 'email', autocomplete: 'off' }"
              placeholder="foo@example.com"
            />
          </NFormItem>
          <!-- In IAM development, we'll show roles multiple selector -->
          <NFormItem
            v-if="isDevelopmentIAM"
            path="roles"
            :label="$t('common.role.self')"
          >
            <NSelect
              v-model:value="state.user.roles"
              multiple
              :options="availableRoles"
              :placeholder="$t('role.select-roles')"
            />
          </NFormItem>
          <!-- TODO(steven): remove this after IAM migrated -->
          <NFormItem v-else path="roles" :label="$t('common.role.self')">
            <NSelect
              :value="state.user.roles[0]"
              :options="availableRoles"
              :placeholder="$t('role.select-role')"
              @change="state.user.roles = [$event]"
            />
          </NFormItem>
          <NFormItem
            v-if="!isCreating"
            path="title"
            :label="$t('common.nickname')"
          >
            <NInput
              v-model:value="state.user.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="foo"
            />
          </NFormItem>
          <div
            v-if="isCreating"
            class="col-span-2 flex justify-start gap-x-2 items-center text-sm text-gray-500"
          >
            <NSwitch
              :value="state.user.userType === UserType.SERVICE_ACCOUNT"
              size="small"
              @update:value="toggleUserServiceAccount($event)"
            />
            <span>
              {{ $t("settings.members.create-as-service-account") }}
              <a
                target="_blank"
                href="https://www.bytebase.com/docs/get-started/terraform?source=console"
              >
                <heroicons-outline:question-mark-circle
                  class="w-4 h-4 inline-block mb-0.5"
                />
              </a>
            </span>
          </div>
        </NForm>
      </template>
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div>
            <NPopconfirm
              v-if="!isCreating"
              :disabled="!allowDeactivate"
              @positive-click="handleArchiveUser"
            >
              <template #trigger>
                <NButton
                  quaternary
                  size="small"
                  :disabled="!allowDeactivate"
                  @click.stop
                >
                  <template #icon>
                    <ArchiveIcon class="w-4 h-auto" />
                  </template>
                  <template #default>
                    {{ $t("settings.members.action.deactivate") }}
                  </template>
                </NButton>
              </template>

              <template #default>
                <div>
                  {{ $t("settings.members.action.deactivate-confirm-title") }}
                </div>
              </template>
            </NPopconfirm>
          </div>

          <div class="flex flex-row items-center justify-end gap-x-3">
            <NButton @click="$emit('close')">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              :loading="state.isRequesting"
              @click="tryCreateOrUpdateUser"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { ArchiveIcon } from "lucide-vue-next";
import {
  NPopconfirm,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NSwitch,
  NSelect,
} from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  getUpdateMaskFromUsers,
  pushNotification,
  useActuatorV1Store,
  useRoleStore,
  useUserStore,
} from "@/store";
import { PresetRoleType, emptyUser } from "@/types";
import {
  UpdateUserRequest,
  User,
  UserType,
} from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { displayRoleTitle, randomString } from "@/utils";

interface LocalState {
  isRequesting: boolean;
  user: User;
}

const serviceAccountEmailSuffix = "@service.bytebase.com";

const props = defineProps<{
  user?: User;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const actuatorStore = useActuatorV1Store();
const userStore = useUserStore();
const state = reactive<LocalState>({
  isRequesting: false,
  user: cloneDeep(props.user) || emptyUser(),
});

const availableRoles = computed(() => {
  const roles = isDevelopmentIAM.value
    ? useRoleStore().roleList.map((role) => role.name)
    : [
        PresetRoleType.WORKSPACE_ADMIN,
        PresetRoleType.WORKSPACE_DBA,
        PresetRoleType.WORKSPACE_MEMBER,
      ];
  return roles.map((role) => ({
    label: displayRoleTitle(role),
    value: role,
  }));
});

const isDevelopmentIAM = computed(() => actuatorStore.serverInfo?.iamGuard);

const isCreating = computed(() => !props.user);

const allowConfirm = computed(() => {
  if (isCreating.value) {
    return state.user.email && state.user.roles.length > 0;
  } else {
    return getUpdateMaskFromUsers(props.user!, state.user).length > 0;
  }
});

const allowDeactivate = computed(() => {
  if (state.user.userType === UserType.SERVICE_ACCOUNT) {
    return true;
  }

  return (
    state.user.state === State.ACTIVE &&
    (hasMoreThanOneOwner.value ||
      !state.user.roles.includes(PresetRoleType.WORKSPACE_ADMIN))
  );
});

const hasMoreThanOneOwner = computed(() => {
  return (
    userStore.userList.filter(
      (user) =>
        user.userType === UserType.USER &&
        user.state === State.ACTIVE &&
        user.roles.includes(PresetRoleType.WORKSPACE_ADMIN)
    ).length > 1
  );
});

const toggleUserServiceAccount = (on: boolean) => {
  state.user.userType = on ? UserType.SERVICE_ACCOUNT : UserType.USER;
};

const handleArchiveUser = () => {
  userStore.archiveUser(props.user!);
  emit("close");
};

const tryCreateOrUpdateUser = async () => {
  if (isCreating.value) {
    if (state.user.userType === UserType.SERVICE_ACCOUNT) {
      state.user.email += serviceAccountEmailSuffix;
    }

    await userStore.createUser({
      ...state.user,
      title: state.user.title || state.user.email,
      password: randomString(20),
    });
  } else {
    // If the user is the only workspace admin, we need to make sure the user is not removing the
    // workspace admin role.
    if (props.user?.roles.includes(PresetRoleType.WORKSPACE_ADMIN)) {
      if (
        !state.user.roles.includes(PresetRoleType.WORKSPACE_ADMIN) &&
        !hasMoreThanOneOwner.value
      ) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("settings.members.tooltip.not-allow-remove"),
        });
        return;
      }
    }

    await userStore.updateUser(
      UpdateUserRequest.fromPartial({
        user: state.user,
        updateMask: getUpdateMaskFromUsers(props.user!, state.user),
      })
    );
  }
  emit("close");
};
</script>
