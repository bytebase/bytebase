<template>
  <main
    class="flex-1 h-full relative pb-8 focus:outline-hidden xl:order-last"
    tabindex="0"
  >
    <NoPermissionPlaceholder
      v-if="!allowGet"
      class="px-4"
      :permissions="['bb.users.get']"
    />
    <div v-else>
      <!-- Profile header -->
      <div>
        <div class="-mt-4 h-32 bg-accent lg:h-48"></div>
        <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <div class="-mt-20 sm:flex sm:items-end sm:gap-x-5">
            <UserAvatar :user="user" size="HUGE" />
            <div
              class="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:gap-x-6 sm:pb-1"
            >
              <div class="mt-6 flex flex-row justify-stretch gap-x-2">
                <template v-if="allowEdit">
                  <template v-if="state.editing">
                    <NButton @click.prevent="cancelEdit">
                      {{ $t("common.cancel") }}
                    </NButton>
                    <NButton
                      type="primary"
                      :disabled="!allowSaveEdit"
                      @click.prevent="saveEdit"
                    >
                      {{ $t("common.save") }}
                    </NButton>
                  </template>
                  <NButton v-else @click.prevent="editUser">
                    {{ $t("common.edit") }}
                  </NButton>
                </template>
              </div>
            </div>
          </div>
          <div class="block mt-6 min-w-0 flex-1">
            <NInput
              v-if="state.editing"
              ref="editNameTextField"
              :input-props="{ autocomplete: 'off' }"
              :value="state.editingUser?.title"
              style="width: 16rem"
              size="large"
              @update:value="updateUser('title', $event)"
            />
            <h1 v-else class="pb-1.5 text-2xl font-bold text-main truncate">
              {{ user.title }}
            </h1>
            <ServiceAccountTag
              v-if="user.userType === UserType.SERVICE_ACCOUNT"
            />
          </div>
        </div>
      </div>

      <!-- Description list -->
      <div class="mt-6 mb-2 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-3">
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.role") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <div
                class="flex flex-row justify-start items-start flex-wrap gap-2"
              >
                <NTag
                  v-for="role in sortRoles(userRoles)"
                  :key="role"
                  size="large"
                >
                  {{ displayRoleTitle(role) }}
                </NTag>
              </div>
              <router-link
                v-if="!hasFeature(PlanFeature.FEATURE_IAM)"
                :to="{
                  name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
                }"
                class="normal-link"
              >
                {{ $t("settings.profile.subscription") }}
              </router-link>
            </dd>
          </div>

          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.email") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <EmailInput
                v-if="state.editing && allowEditEmail"
                v-model:value="state.editingUser!.email"
              />
              <template v-else>
                {{ user.email }}
              </template>
            </dd>
          </div>

          <div v-if="user.userType === UserType.USER" class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.phone") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <NInput
                v-if="state.editing"
                :value="state.editingUser?.phone"
                :placeholder="$t('settings.profile.phone-tips')"
                :input-props="{ autocomplete: 'off', type: 'tel' }"
                @update:value="updateUser('phone', $event)"
              />
              <template v-else>
                {{ user.phone }}
              </template>
            </dd>
          </div>

          <div v-if="state.editing" class="col-span-2">
            <UserPassword
              v-if="state.editingUser"
              ref="userPasswordRef"
              v-model:password="state.editingUser!.password"
              v-model:password-confirm="state.passwordConfirm"
              :password-restriction="passwordRestrictionSetting"
            />
          </div>
        </dl>
      </div>

      <!-- 2FA setting section -->
      <div
        v-if="allowEdit"
        class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 border-t mt-16 pt-8 pb-4"
      >
        <div class="w-full flex flex-row justify-between items-center">
          <span
            class="text-lg font-medium flex flex-row justify-start items-center"
          >
            {{ $t("two-factor.self") }}
            <FeatureBadge :feature="PlanFeature.FEATURE_TWO_FA" class="ml-2" />
          </span>
          <div class="flex gap-x-2">
            <NButton v-if="isMFAEnabled" type="error" @click="disable2FA">
              {{ $t("common.disable") }}
            </NButton>
            <NButton v-if="user.email === currentUser.email" @click="enable2FA">
              {{ isMFAEnabled ? $t("common.edit") : $t("common.enable") }}
            </NButton>
          </div>
        </div>
        <p class="mt-4 text-sm text-gray-500">
          {{ $t("two-factor.description") }}
          <LearnMoreLink
            class="ml-1"
            url="https://docs.bytebase.com/administration/2fa?source=console"
          />
        </p>
        <template v-if="showRegenerateRecoveryCodes">
          <div class="w-full flex flex-row justify-between items-center mt-8">
            <span class="text-lg font-medium">
              {{ $t("two-factor.recovery-codes.self") }}
            </span>
            <div v-if="!state.showRegenerateRecoveryCodesView" class="relative">
              <NDropdown
                trigger="click"
                :options="dropDownOptions"
                placement="bottom-end"
              >
                <MiniActionButton size="small">
                  <EllipsisIcon class="w-8" />
                </MiniActionButton>
              </NDropdown>
            </div>
          </div>
          <p class="mt-4 text-sm text-gray-500">
            {{ $t("two-factor.recovery-codes.description") }}
          </p>
          <RegenerateRecoveryCodesView
            v-if="state.showRegenerateRecoveryCodesView"
            :recovery-codes="currentUser.tempRecoveryCodes"
            @close="state.showRegenerateRecoveryCodesView = false"
          />
        </template>
      </div>
    </div>
  </main>

  <FeatureModal
    :feature="PlanFeature.FEATURE_TWO_FA"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-model:show="state.showDisable2FAConfirmModal"
    :title="$t('two-factor.disable.self')"
    :description="$t('two-factor.disable.description')"
    :positive-button-props="{
      type: 'error',
    }"
    @confirm="handleDisable2FA"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { computedAsync, useTitle } from "@vueuse/core";
import { cloneDeep, isEqual } from "lodash-es";
import { EllipsisIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NInput, NTag } from "naive-ui";
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import EmailInput from "@/components/EmailInput.vue";
import { FeatureModal } from "@/components/FeatureGuard";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import RegenerateRecoveryCodesView from "@/components/RegenerateRecoveryCodesView.vue";
import { ActionConfirmModal } from "@/components/SchemaEditorLite";
import UserPassword from "@/components/User/Settings/UserPassword.vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { MiniActionButton } from "@/components/v2";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import {
  SETTING_ROUTE_PROFILE_TWO_FACTOR,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import {
  featureToRef,
  hasFeature,
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useSettingV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  ALL_USERS_USER_EMAIL,
  SYSTEM_BOT_USER_NAME,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import { displayRoleTitle, hasWorkspacePermissionV2, sortRoles } from "@/utils";

interface LocalState {
  editing: boolean;
  editingUser?: User;
  passwordConfirm: string;
  showFeatureModal: boolean;
  showDisable2FAConfirmModal: boolean;
  showRegenerateRecoveryCodesView: boolean;
}

const props = defineProps<{
  principalEmail?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const actuatorStore = useActuatorV1Store();
const settingV1Store = useSettingV1Store();
const currentUser = useCurrentUserV1();
const userStore = useUserStore();
const workspaceStore = useWorkspaceV1Store();

const state = reactive<LocalState>({
  editing: false,
  passwordConfirm: "",
  showFeatureModal: false,
  showDisable2FAConfirmModal: false,
  showRegenerateRecoveryCodesView: false,
});

const editNameTextField = ref<InstanceType<typeof NInput>>();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const passwordRestrictionSetting = computed(
  () => settingV1Store.passwordRestriction
);

const keyboardHandler = (e: KeyboardEvent) => {
  if (state.editing) {
    if (e.code === "Escape") {
      cancelEdit();
    } else if (e.code === "Enter" && e.metaKey) {
      if (allowSaveEdit.value) {
        saveEdit();
      }
    }
  }
};

onMounted(async () => {
  document.addEventListener("keydown", keyboardHandler);
});

onUnmounted(() => {
  document.removeEventListener("keydown", keyboardHandler);
});

const has2FAFeature = featureToRef(PlanFeature.FEATURE_TWO_FA);

const isMFAEnabled = computed(() => {
  return user.value.mfaEnabled;
});

// only user can regenerate their recovery-codes.
const showRegenerateRecoveryCodes = computed(() => {
  return user.value.mfaEnabled && user.value.name === currentUser.value.name;
});

const user = computedAsync(() => {
  if (props.principalEmail) {
    return userStore.getOrFetchUserByIdentifier(props.principalEmail);
  }
  return currentUser.value;
}, unknownUser());

const userRoles = computed(() => {
  return [...workspaceStore.getWorkspaceRolesByEmail(user.value.email)];
});

const isSelf = computed(() => currentUser.value.name === user.value.name);

const allowGet = computed(
  () => isSelf.value || hasWorkspacePermissionV2("bb.users.get")
);

// User can change her own info.
// Besides, owner can also change anyone's info. This is for resetting password in case user forgets.
const allowEdit = computed(() => {
  if (
    user.value.name === SYSTEM_BOT_USER_NAME ||
    user.value.email === ALL_USERS_USER_EMAIL ||
    user.value.userType !== UserType.USER
  ) {
    return false;
  }
  if (user.value.state !== State.ACTIVE) {
    return false;
  }
  return isSelf.value || hasWorkspacePermissionV2("bb.users.update");
});

// Only users with bb.users.updateEmail permission can change email.
const allowEditEmail = computed(() => {
  return hasWorkspacePermissionV2("bb.users.updateEmail");
});

const allowSaveEdit = computed(() => {
  return (
    !isEqual(user.value, state.editingUser) &&
    !userPasswordRef.value?.passwordHint &&
    !userPasswordRef.value?.passwordMismatch
  );
});

useRouteChangeGuard(computed(() => state.editing && allowSaveEdit.value));

const updateUser = <K extends keyof User>(field: K, value: User[K]) => {
  if (!state.editingUser) return;

  state.editingUser[field] = value;
};

const editUser = () => {
  state.editingUser = cloneDeep(user.value);
  state.editing = true;
  state.passwordConfirm = "";

  nextTick(() => editNameTextField.value?.focus());
};

const cancelEdit = () => {
  state.editingUser = undefined;
  state.editing = false;
};

const saveEdit = async () => {
  const userPatch = state.editingUser;
  if (!userPatch) return;

  const emailChanged = userPatch.email !== user.value.email;
  const updateMaskPaths: string[] = [];

  if (userPatch.title !== user.value.title) {
    updateMaskPaths.push("title");
  }
  if (userPatch.phone !== user.value.phone) {
    updateMaskPaths.push("phone");
  }
  if (userPatch.password !== "") {
    updateMaskPaths.push("password");
  }

  try {
    // Update email using dedicated UpdateEmail API if changed
    if (emailChanged) {
      await userStore.updateEmail(user.value.email, userPatch.email);
    }

    // Update other fields using UpdateUser API if any changed
    if (updateMaskPaths.length > 0) {
      await userStore.updateUser(
        create(UpdateUserRequestSchema, {
          user: userPatch,
          updateMask: create(FieldMaskSchema, {
            paths: updateMaskPaths,
          }),
          regenerateRecoveryCodes: false,
          regenerateTempMfaSecret: false,
        })
      );
    }
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message || "Failed to update user",
    });
    return;
  }

  state.editingUser = undefined;
  state.editing = false;

  // If we update email, we need to redirect to the new email.
  if (emailChanged && props.principalEmail) {
    router.replace({
      name: WORKSPACE_ROUTE_USER_PROFILE,
      params: {
        principalEmail: userPatch.email,
      },
    });
  }
};

const enable2FA = () => {
  if (!has2FAFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  router.push({ name: SETTING_ROUTE_PROFILE_TWO_FACTOR });
};

const disable2FA = () => {
  if (
    actuatorStore.serverInfo?.require2fa &&
    !hasWorkspacePermissionV2("bb.policies.update")
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("two-factor.messages.cannot-disable"),
    });
  } else {
    state.showDisable2FAConfirmModal = true;
  }
};

const handleDisable2FA = async () => {
  await userStore.updateUser(
    create(UpdateUserRequestSchema, {
      user: {
        name: user.value.name,
        mfaEnabled: false,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["mfa_enabled"],
      }),
    })
  );
  state.showDisable2FAConfirmModal = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("two-factor.messages.2fa-disabled"),
  });
};

const dropDownOptions = computed((): DropdownOption[] => [
  {
    label: t("common.regenerate"),
    key: "regenerate",
    props: {
      onClick: () => {
        state.showRegenerateRecoveryCodesView = true;
      },
    },
  },
]);

useTitle(computed(() => user.value.title));
</script>
