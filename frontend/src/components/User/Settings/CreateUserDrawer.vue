<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="
        isCreating
          ? $t('settings.members.add-user')
          : $t('settings.members.update-user')
      "
    >
      <template #default>
        <NForm>
          <div
            v-if="isCreating && !hideServiceAccount"
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
            v-if="state.user.userType !== UserType.SERVICE_ACCOUNT"
            :label="$t('common.name')"
          >
            <NInput
              v-model:value="state.user.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="Foo"
            />
          </NFormItem>
          <NFormItem :label="$t('common.email')" required>
            <div
              v-if="state.user.userType === UserType.SERVICE_ACCOUNT"
              class="w-full flex flex-col items-start"
            >
              <EmailInput v-model:value="state.user.email" />
            </div>
            <EmailInput
              v-else
              v-model:value="state.user.email"
              :readonly="disallowEditUser"
              :domain="workspaceDomain"
            />
          </NFormItem>
          <NFormItem :label="$t('settings.members.table.roles')">
            <div class="w-full space-y-1">
              <span class="textinfolabel text-sm">
                {{ $t("role.default-workspace-role") }}
              </span>
              <RoleSelect v-model:value="state.roles" :multiple="true" />
            </div>
          </NFormItem>
          <template v-if="state.user.userType === UserType.USER">
            <NFormItem :label="$t('settings.profile.phone')">
              <div class="w-full space-y-1">
                <span class="textinfolabel text-sm">
                  {{ $t("settings.profile.phone-tips") }}
                </span>
                <NInput
                  v-model:value="state.user.phone"
                  type="text"
                  :input-props="{
                    type: 'tel',
                    autocomplete: 'new-password',
                  }"
                />
              </div>
            </NFormItem>

            <UserPassword
              ref="userPasswordRef"
              v-model:password="state.user.password"
              v-model:password-confirm="state.passwordConfirm"
              :password-restriction="passwordRestrictionSetting"
            />
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
import { cloneDeep, head, isEqual, isUndefined } from "lodash-es";
import { ArchiveIcon } from "lucide-vue-next";
import {
  NPopconfirm,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NRadioGroup,
  NRadio,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  getUpdateMaskFromUsers,
  pushNotification,
  useAppFeature,
  useSettingV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import { PresetRoleType, emptyUser } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  UpdateUserRequest,
  UserType,
  User,
} from "@/types/proto/v1/user_service";
import UserPassword from "./UserPassword.vue";

interface LocalState {
  isRequesting: boolean;
  user: User;
  roles: string[];
  passwordConfirm: string;
}

const props = defineProps<{
  user?: User;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created"): void;
}>();

const workspaceStore = useWorkspaceV1Store();

const userRolesFromProps = computed(() => {
  return workspaceStore.getWorkspaceRolesByEmail(props.user?.email ?? "");
});

const initRoles = () => {
  return [...userRolesFromProps.value].filter(
    (r) => r !== PresetRoleType.WORKSPACE_MEMBER
  );
};

const initUser = () => {
  return props.user ? cloneDeep(props.user) : emptyUser();
};

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const userStore = useUserStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const hideServiceAccount = useAppFeature(
  "bb.feature.members.hide-service-account"
);

const state = reactive<LocalState>({
  isRequesting: false,
  user: initUser(),
  roles: initRoles(),
  passwordConfirm: "",
});

const passwordRestrictionSetting = computed(
  () => settingV1Store.passwordRestriction
);

const workspaceDomain = computed(() => {
  if (!settingV1Store.workspaceProfileSetting?.enforceIdentityDomain) {
    return undefined;
  }
  return head(settingV1Store.workspaceProfileSetting?.domains);
});

const isCreating = computed(() => !props.user);

const rolesChanged = computed(() => {
  if (isCreating.value) {
    return true;
  }

  return !isUndefined(state.roles) && !isEqual(initRoles(), state.roles);
});

const disallowEditUser = computed(() => !!props.user?.profile?.source);

const allowConfirm = computed(() => {
  if (!state.user.email) {
    return false;
  }
  if (userPasswordRef.value?.passwordHint) {
    return false;
  }
  if (userPasswordRef.value?.passwordMismatch) {
    return false;
  }

  if (
    !isCreating.value &&
    getUpdateMaskFromUsers(props.user!, state.user).length == 0 &&
    !rolesChanged.value
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
      !state.roles.includes(PresetRoleType.WORKSPACE_ADMIN))
  );
});

const hasMoreThanOneOwner = computed(() => {
  return (
    (workspaceStore.roleMapToUsers.get(PresetRoleType.WORKSPACE_ADMIN)?.size ??
      0) > 1
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
    const createdUser = await userStore.createUser({
      ...state.user,
      title: state.user.title || extractUserTitle(state.user.email),
      password: state.user.password,
    });
    if (state.roles.length > 0) {
      await workspaceStore.patchIamPolicy([
        {
          member: `user:${createdUser.email}`,
          roles: state.roles,
        },
      ]);
    }
    emit("created");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
  } else {
    // If the user is the only workspace admin, we need to make sure the user is not removing the
    // workspace admin role.
    if (userRolesFromProps.value.has(PresetRoleType.WORKSPACE_ADMIN)) {
      if (
        !state.roles.includes(PresetRoleType.WORKSPACE_ADMIN) &&
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
    if (rolesChanged.value) {
      await workspaceStore.patchIamPolicy([
        {
          member: `user:${state.user.email}`,
          roles: state.roles,
        },
      ]);
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
  emit("close");
};
</script>
