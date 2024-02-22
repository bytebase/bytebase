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
        <NForm>
          <div
            v-if="isCreating"
            class="w-full mb-4 flex flex-row justify-start items-center"
          >
            <span class="mr-2 text-sm">{{ $t("common.type") }}</span>
            <NRadioGroup v-model:value="state.user.userType" name="userType">
              <NRadio :value="UserType.USER" :label="$t('common.user')" />
              <NRadio
                :value="UserType.SERVICE_ACCOUNT"
                :label="$t('settings.members.service-account')"
              />
            </NRadioGroup>
            <a
              href="https://www.bytebase.com/docs/get-started/terraform?source=console"
              target="_blank"
            >
              <heroicons-outline:question-mark-circle class="w-4 h-4" />
            </a>
          </div>
          <NFormItem
            v-if="
              (isCreating &&
                state.user.userType !== UserType.SERVICE_ACCOUNT) ||
              !isCreating
            "
            :label="$t('common.name')"
          >
            <NInput
              v-model:value="state.user.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="foo"
            />
          </NFormItem>
          <NFormItem :label="$t('common.email')">
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
          <NFormItem :label="$t('settings.members.table.roles')">
            <div class="w-full">
              <NSelect
                v-model:value="state.user.roles"
                multiple
                :options="availableRoleOptions"
                :placeholder="$t('role.select-roles')"
              />
              <p
                v-if="state.user.roles.length > 0 && !hasWorkspaceRole"
                class="textinfolabel mt-1 !text-red-600"
              >
                {{ $t("settings.members.workspace-role-at-least-one") }}
              </p>
            </div>
          </NFormItem>
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
  NSelect,
  NRadioGroup,
  NRadio,
} from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  getUpdateMaskFromUsers,
  pushNotification,
  useRoleStore,
  useUserStore,
} from "@/store";
import { PRESET_WORKSPACE_ROLES, PresetRoleType, emptyUser } from "@/types";
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
const userStore = useUserStore();
const state = reactive<LocalState>({
  isRequesting: false,
  user: cloneDeep(props.user) || emptyUser(),
});

const availableRoleOptions = computed(() => {
  const roles = useRoleStore().roleList.map((role) => role.name);
  return roles.map((role) => ({
    label: displayRoleTitle(role),
    value: role,
  }));
});

const hasWorkspaceRole = computed(() => {
  return state.user.roles.some((role) => PRESET_WORKSPACE_ROLES.includes(role));
});

const isCreating = computed(() => !props.user);

const allowConfirm = computed(() => {
  if (
    !state.user.email ||
    state.user.roles.length === 0 ||
    !hasWorkspaceRole.value
  ) {
    return false;
  }
  if (
    !isCreating.value &&
    getUpdateMaskFromUsers(props.user!, state.user).length == 0
  ) {
    return false;
  }

  return true;
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

const extractUserTitle = (email: string): string => {
  const atIndex = email.indexOf("@");
  if (atIndex !== -1) {
    return email.substring(0, atIndex);
  }
  // If there is no @, we just return the email as title.
  return email;
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
      title: state.user.title || extractUserTitle(state.user.email),
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
