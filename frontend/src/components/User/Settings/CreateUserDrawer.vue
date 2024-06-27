<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
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
            <NRadioGroup v-model:value="state.user.userType">
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
          <NFormItem :label="$t('common.email')" required>
            <div
              v-if="
                isCreating && state.user.userType === UserType.SERVICE_ACCOUNT
              "
              class="w-full flex flex-col items-start"
            >
              <NInput
                v-model:value="state.user.email"
                :input-props="{ type: 'text', autocomplete: 'new-password' }"
                placeholder="foo"
              />
              <span class="mt-1 textinfolabel">
                {{ serviceAccountEmailSuffix }}
              </span>
            </div>
            <EmailInput
              v-else
              v-model:value="state.user.email"
              :domain="workspaceDomain"
            />
          </NFormItem>
          <NFormItem :label="$t('settings.members.table.roles')">
            <div class="w-full space-y-1">
              <span class="textinfolabel text-sm">
                {{ $t("role.default-workspace-role") }}
              </span>
              <NSelect
                v-model:value="state.user.roles"
                multiple
                :options="availableRoleOptions"
                :placeholder="$t('role.select-roles')"
              />
            </div>
          </NFormItem>
          <template v-if="state.user.userType === UserType.USER">
            <NFormItem :label="$t('settings.profile.phone')">
              <NInput
                v-model:value="state.user.phone"
                type="text"
                :input-props="{
                  type: 'tel',
                  autocomplete: 'new-password',
                }"
              />
            </NFormItem>
            <NFormItem :label="$t('settings.profile.password')">
              <NInput
                v-model:value="state.user.password"
                type="password"
                :input-props="{ autocomplete: 'new-password' }"
                :placeholder="$t('common.sensitive-placeholder')"
              />
            </NFormItem>
            <NFormItem :label="$t('settings.profile.password-confirm')">
              <div class="w-full flex flex-col justify-start items-start">
                <NInput
                  v-model:value="state.passwordConfirm"
                  type="password"
                  :input-props="{ autocomplete: 'new-password' }"
                  :placeholder="
                    $t('settings.profile.password-confirm-placeholder')
                  "
                />
                <span
                  v-if="passwordMismatch"
                  class="text-error text-sm mt-1 pl-1"
                  >{{ $t("settings.profile.password-mismatch") }}</span
                >
              </div>
            </NFormItem>
          </template>
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
import { cloneDeep, head, isEmpty } from "lodash-es";
import { ArchiveIcon } from "lucide-vue-next";
import type { SelectGroupOption, SelectOption } from "naive-ui";
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
import EmailInput from "@/components/EmailInput.vue";
import {
  getUpdateMaskFromUsers,
  pushNotification,
  useRoleStore,
  useSettingV1Store,
  useUserStore,
} from "@/store";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
  emptyUser,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { UpdateUserRequest, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { displayRoleTitle, randomString } from "@/utils";

interface LocalState {
  isRequesting: boolean;
  user: User;
  passwordConfirm: string;
}

const serviceAccountEmailSuffix = "@service.bytebase.com";

const props = defineProps<{
  user?: User;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const initUser = () => {
  const user = props.user ? cloneDeep(props.user) : emptyUser();
  return {
    ...user,
    roles: user.roles.filter((r) => r !== PresetRoleType.WORKSPACE_MEMBER),
  };
};

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const userStore = useUserStore();

const state = reactive<LocalState>({
  isRequesting: false,
  user: initUser(),
  passwordConfirm: "",
});

const availableRoleOptions = computed(
  (): (SelectOption | SelectGroupOption)[] => {
    const roleGroups = [
      {
        type: "group",
        key: "workspace-roles",
        label: t("role.workspace-roles"),
        children: PRESET_WORKSPACE_ROLES.filter(
          (role) => role !== PresetRoleType.WORKSPACE_MEMBER
        ).map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
      {
        type: "group",
        key: "project-roles",
        label: `${t("role.project-roles.self")} (${t(
          "role.project-roles.apply-to-all-projects"
        )})`,
        children: PRESET_PROJECT_ROLES.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      },
    ];
    const customRoles = useRoleStore()
      .roleList.map((role) => role.name)
      .filter((role) => !PRESET_ROLES.includes(role));
    if (customRoles.length > 0) {
      roleGroups.push({
        type: "group",
        key: "custom-roles",
        label: `${t("role.custom-roles")} (${t(
          "role.project-roles.apply-to-all-projects"
        )})`,
        children: customRoles.map((role) => ({
          label: displayRoleTitle(role),
          value: role,
        })),
      });
    }
    return roleGroups;
  }
);

const workspaceDomain = computed(() => {
  if (!settingV1Store.workspaceProfileSetting?.enforceIdentityDomain) {
    return undefined;
  }
  return head(settingV1Store.workspaceProfileSetting?.domains);
});

const isCreating = computed(() => !props.user);

const passwordMismatch = computed(() => {
  return (
    !isEmpty(state.user?.password) &&
    state.user?.password !== state.passwordConfirm
  );
});

const allowConfirm = computed(() => {
  if (!state.user.email) {
    return false;
  }
  if (
    !isCreating.value &&
    (passwordMismatch.value ||
      getUpdateMaskFromUsers(props.user!, state.user).length == 0)
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

const handleArchiveUser = async () => {
  await userStore.archiveUser(props.user!);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.archived"),
  });
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
      password: state.user.password || randomString(20),
    });
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("common.created"),
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
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
  emit("close");
};
</script>
