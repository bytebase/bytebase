<template>
  <main
    class="flex-1 h-full relative pb-8 focus:outline-none xl:order-last"
    tabindex="0"
  >
    <!-- Profile header -->
    <div>
      <div class="h-32 w-full bg-accent lg:h-48"></div>
      <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="-mt-20 sm:flex sm:items-end sm:space-x-5">
          <UserAvatar :user="user" size="HUGE" />
          <div
            class="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:space-x-6 sm:pb-1"
          >
            <div class="mt-6 flex flex-row justify-stretch space-x-4">
              <template v-if="allowEdit">
                <template v-if="state.editing">
                  <NButton @click.prevent="cancelEdit">
                    {{ $t("common.cancel") }}
                  </NButton>
                  <NButton :disabled="!allowSaveEdit" @click.prevent="saveEdit">
                    <template #icon>
                      <heroicons-solid:save
                        class="h-5 w-5 text-control-light"
                      />
                    </template>
                    {{ $t("common.save") }}
                  </NButton>
                </template>
                <NButton v-else @click.prevent="editUser">
                  <template #icon>
                    <heroicons-solid:pencil
                      class="h-5 w-5 text-control-light"
                    />
                  </template>
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
          <span
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
          >
            {{ $t("settings.members.service-account") }}
          </span>
        </div>
      </div>
    </div>

    <!-- Description list -->
    <div
      v-if="user.userType === UserType.USER"
      class="mt-6 mb-2 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8"
    >
      <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-3">
        <div class="sm:col-span-1">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("settings.profile.role") }}
          </dt>
          <dd class="mt-1 text-sm text-main">
            <router-link :to="'/setting/member'" class="normal-link capitalize">
              {{ roleNameV1(user.userRole) }}
            </router-link>
            <router-link
              v-if="!hasRBACFeature"
              :to="'/setting/subscription'"
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
            <NInput
              v-if="state.editing"
              size="large"
              :value="state.editingUser?.email"
              :input-props="{ autocomplete: 'off', type: 'email' }"
              @update:value="updateUser('email', $event)"
            />
            <template v-else>
              {{ user.email }}
            </template>
          </dd>
        </div>

        <div class="sm:col-span-1">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("settings.profile.phone") }}
          </dt>
          <dd class="mt-1 text-sm text-main">
            <PhoneNumberInput
              v-if="state.editing"
              :value="state.editingUser?.phone || ''"
              @update="(value: string) => updateUser('phone', value)"
            />
            <template v-else>
              {{ user.phone }}
            </template>
          </dd>
        </div>

        <template v-if="state.editing">
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.password") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <NInput
                type="password"
                size="large"
                :placeholder="$t('common.sensitive-placeholder')"
                :value="state.editingUser?.password"
                :input-props="{ autocomplete: 'off' }"
                @update:value="updateUser('password', $event)"
              />
            </dd>
          </div>

          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.password-confirm") }}
              <span v-if="passwordMismatch" class="text-error">{{
                $t("settings.profile.password-mismatch")
              }}</span>
            </dt>
            <dd class="mt-1 text-sm text-main">
              <NInput
                type="password"
                size="large"
                :placeholder="
                  $t('settings.profile.password-confirm-placeholder')
                "
                :value="state.passwordConfirm"
                :input-props="{ autocomplete: 'off' }"
                @update:value="state.passwordConfirm = $event"
              />
            </dd>
          </div>
        </template>
      </dl>
    </div>

    <!-- 2FA setting section -->
    <div
      v-if="showMFAConfig"
      class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 border-t mt-16 pt-8 pb-4"
    >
      <div class="w-full flex flex-row justify-between items-center">
        <span
          class="text-lg font-medium flex flex-row justify-start items-center"
        >
          {{ $t("two-factor.self") }}
          <FeatureBadge :feature="'bb.feature.2fa'" custom-class="ml-2" />
        </span>
        <div class="space-x-2">
          <NButton @click="enable2FA">
            {{ isMFAEnabled ? $t("common.edit") : $t("common.enable") }}
          </NButton>
          <NButton v-if="isMFAEnabled" @click="disable2FA">
            {{ $t("common.disable") }}
          </NButton>
        </div>
      </div>
      <p class="mt-4 text-sm text-gray-500">
        {{ $t("two-factor.description") }}
        <LearnMoreLink
          class="ml-1"
          url="https://www.bytebase.com/docs/administration/2fa?source=console"
        />
      </p>
      <template v-if="isMFAEnabled">
        <div class="w-full flex flex-row justify-between items-center mt-8">
          <span class="text-lg font-medium">{{
            $t("two-factor.recovery-codes.self")
          }}</span>
          <div v-if="!state.showRegenerateRecoveryCodesView" class="relative">
            <heroicons-outline:ellipsis-horizontal
              class="w-8 p-1 h-auto cursor-pointer hover:bg-gray-100 rounded"
              @click.prevent="recoveryCodesMenu.toggle()"
              @contextmenu.capture.prevent="recoveryCodesMenu.toggle()"
            />
            <BBContextMenu
              ref="recoveryCodesMenu"
              class="origin-top-left mt-1 w-32 shadow"
            >
              <div class="py-1">
                <a
                  class="menu-item"
                  role="menuitem"
                  @click="state.showRegenerateRecoveryCodesView = true"
                >
                  {{ $t("common.regenerate") }}
                </a>
              </div>
            </BBContextMenu>
          </div>
        </div>
        <p class="mt-4 text-sm text-gray-500">
          {{ $t("two-factor.recovery-codes.description") }}
        </p>
        <RegenerateRecoveryCodesView
          v-if="state.showRegenerateRecoveryCodesView"
          :recovery-codes="authStore.currentUser.recoveryCodes"
          @close="state.showRegenerateRecoveryCodesView = false"
        />
      </template>
    </div>
  </main>

  <FeatureModal
    feature="bb.feature.2fa"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-if="state.showDisable2FAConfirmModal"
    :title="$t('two-factor.disable.self')"
    :description="$t('two-factor.disable.description')"
    :style="'danger'"
    @close="state.showDisable2FAConfirmModal = false"
    @confirm="handleDisable2FA"
  />
</template>

<script lang="ts" setup>
import { nextTick, computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { NButton, NInput } from "naive-ui";
import { useI18n } from "vue-i18n";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { useRouter } from "vue-router";

import { unknownUser } from "../types";
import { hasWorkspacePermissionV1, roleNameV1 } from "../utils";
import {
  featureToRef,
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useCurrentUserV1,
  useUserStore,
} from "@/store";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import {
  UpdateUserRequest,
  User,
  UserType,
} from "@/types/proto/v1/auth_service";
import RegenerateRecoveryCodesView from "@/components/RegenerateRecoveryCodesView.vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import PhoneNumberInput from "@/components/v2/Form/PhoneNumberInput.vue";

interface LocalState {
  editing: boolean;
  editingUser?: User;
  passwordConfirm: string;
  showFeatureModal: boolean;
  showDisable2FAConfirmModal: boolean;
  showRegenerateRecoveryCodesView: boolean;
}

const props = defineProps<{
  principalId?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const actuatorStore = useActuatorV1Store();
const authStore = useAuthStore();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const state = reactive<LocalState>({
  editing: false,
  passwordConfirm: "",
  showFeatureModal: false,
  showDisable2FAConfirmModal: false,
  showRegenerateRecoveryCodesView: false,
});

const editNameTextField = ref<InstanceType<typeof NInput>>();
const recoveryCodesMenu = ref();

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

const hasRBACFeature = featureToRef("bb.feature.rbac");
const has2FAFeature = featureToRef("bb.feature.2fa");

const isMFAEnabled = computed(() => {
  return authStore.currentUser.mfaEnabled;
});

const user = computed(() => {
  if (props.principalId) {
    return userStore.getUserById(String(props.principalId)) ?? unknownUser();
  }
  return currentUserV1.value;
});

const showMFAConfig = computed(() => {
  // Only show MFA config for the user themselves.
  return user.value.name === currentUserV1.value.name;
});

const passwordMismatch = computed(() => {
  return (
    !isEmpty(state.editingUser?.password) &&
    state.editingUser?.password !== state.passwordConfirm
  );
});

// User can change her own info.
// Besides, owner can also change anyone's info. This is for resetting password in case user forgets.
const allowEdit = computed(() => {
  return (
    currentUserV1.value.name === user.value.name ||
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-member",
      currentUserV1.value.userRole
    )
  );
});

const allowSaveEdit = computed(() => {
  return (
    !isEqual(user.value, state.editingUser) &&
    (state.passwordConfirm === "" ||
      state.passwordConfirm === state.editingUser?.password)
  );
});

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

  const updateMask: string[] = [];
  if (userPatch.title !== user.value.title) {
    updateMask.push("title");
  }
  if (userPatch.email !== user.value.email) {
    updateMask.push("email");
  }
  if (userPatch.phone !== user.value.phone) {
    updateMask.push("phone");
  }
  if (userPatch.password !== "") {
    updateMask.push("password");
  }
  try {
    await userStore.updateUser({
      user: userPatch,
      updateMask,
      regenerateRecoveryCodes: false,
      regenerateTempMfaSecret: false,
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as any).details || "Failed to update user",
    });
    return;
  }
  await useAuthStore().refreshUserIfNeeded(currentUserV1.value.name);

  state.editingUser = undefined;
  state.editing = false;
};

const enable2FA = () => {
  if (!has2FAFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  router.push({ name: "setting.profile.two-factor" });
};

const disable2FA = () => {
  if (actuatorStore.serverInfo?.require2fa) {
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
  const user = authStore.currentUser;
  await userStore.updateUser(
    UpdateUserRequest.fromPartial({
      user: {
        name: user.name,
        mfaEnabled: false,
      },
      updateMask: ["mfa_enabled"],
    })
  );
  await authStore.refreshUserIfNeeded(user.name);
  state.showDisable2FAConfirmModal = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("two-factor.messages.2fa-disabled"),
  });
};
</script>
