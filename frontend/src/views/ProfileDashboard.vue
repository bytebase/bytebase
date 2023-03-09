<template>
  <main
    class="flex-1 h-full relative z-0 overflow-auto pb-8 focus:outline-none xl:order-last"
    tabindex="0"
  >
    <!-- Profile header -->
    <div>
      <div class="h-32 w-full bg-accent lg:h-48"></div>
      <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="-mt-20 sm:flex sm:items-end sm:space-x-5">
          <PrincipalAvatar :principal="principal" :size="'HUGE'" />
          <div
            class="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:space-x-6 sm:pb-1"
          >
            <div class="mt-6 flex flex-row justify-stretch space-x-4">
              <template v-if="allowEdit">
                <template v-if="state.editing">
                  <button
                    type="button"
                    class="btn-normal"
                    @click.prevent="cancelEdit"
                  >
                    {{ $t("common.cancel") }}
                  </button>
                  <button
                    type="button"
                    class="btn-normal"
                    :disabled="!allowSaveEdit"
                    @click.prevent="saveEdit"
                  >
                    <heroicons-solid:save
                      class="-ml-1 mr-2 h-5 w-5 text-control-light"
                    />
                    <span>{{ $t("common.save") }}</span>
                  </button>
                </template>
                <button
                  v-else
                  type="button"
                  class="btn-normal"
                  @click.prevent="editUser"
                >
                  <heroicons-solid:pencil
                    class="-ml-1 mr-2 h-5 w-5 text-control-light"
                  />
                  <span>{{ $t("common.edit") }}</span>
                </button>
              </template>
            </div>
          </div>
        </div>
        <div class="block mt-6 min-w-0 flex-1">
          <input
            v-if="state.editing"
            id="name"
            ref="editNameTextField"
            required
            autocomplete="off"
            name="name"
            type="text"
            class="textfield"
            :value="state.editingPrincipal?.name"
            @input="(e: any)=>updatePrincipal('name', e.target.value)"
          />
          <h1 v-else class="pb-1.5 text-2xl font-bold text-main truncate">
            {{ principal.name }}
          </h1>
          <span
            v-if="principal.type === 'SERVICE_ACCOUNT'"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
          >
            {{ $t("settings.members.service-account") }}
          </span>
        </div>
      </div>
    </div>

    <!-- Description list -->
    <div
      v-if="principal.type === 'END_USER'"
      class="mt-6 mb-2 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8"
    >
      <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
        <div class="sm:col-span-1">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("settings.profile.role") }}
          </dt>
          <dd class="mt-1 text-sm text-main">
            <router-link
              :to="'/setting/member'"
              class="normal-link capitalize"
              >{{
                $t(`common.role.${principal.role.toLowerCase()}`)
              }}</router-link
            >
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
            <input
              v-if="state.editing"
              id="email"
              required
              autocomplete="off"
              name="email"
              type="text"
              class="textfield"
              :value="state.editingPrincipal?.email"
              @input="(e: any)=>updatePrincipal('email', e.target.value)"
            />
            <template v-else>
              {{ principal.email }}
            </template>
          </dd>
        </div>

        <template v-if="state.editing">
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-control-light">
              {{ $t("settings.profile.password") }}
            </dt>
            <dd class="mt-1 text-sm text-main">
              <input
                id="password"
                name="password"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                :placeholder="$t('common.sensitive-placeholder')"
                :value="state.editingPrincipal?.password"
                @input="(e: any) => updatePrincipal('password', e.target.value)"
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
              <input
                id="password-confirm"
                name="password-confirm"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                :placeholder="
                  $t('settings.profile.password-confirm-placeholder')
                "
                :value="state.passwordConfirm"
                @input="(e: any) => state.passwordConfirm = e.target.value"
              />
            </dd>
          </div>
        </template>
      </dl>
    </div>

    <!-- 2FA setting section -->
    <div
      v-if="isDev"
      class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 border-t mt-16 pt-8 pb-4"
    >
      <div class="w-full flex flex-row justify-between items-center">
        <span
          class="text-lg font-medium flex flex-row justify-start items-center"
        >
          {{ $t("two-factor.self") }}
          <FeatureBadge :feature="'bb.feature.2fa'" class="ml-2 text-accent" />
        </span>
        <BBSwitch
          :value="isMFAEnabled"
          @toggle="handle2FAEnableStatusChanged"
        />
      </div>
      <p class="mt-4 text-sm text-gray-500">
        {{ $t("two-factor.description") }}
        <!-- TODO(steven): update the docs link -->
        <LearnMoreLink class="ml-1" url="" />
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
    v-if="state.showFeatureModal"
    feature="bb.feature.2fa"
    @cancel="state.showFeatureModal = false"
  />

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-if="state.showDisable2FAConfirmModal"
    :title="$t('two-factor.disable.self')"
    :description="$t('two-factor.disable.description')"
    @close="state.showDisable2FAConfirmModal = false"
    @confirm="handleDisable2FA"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { nextTick, computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { PrincipalPatch } from "../types";
import { hasWorkspacePermission } from "../utils";
import {
  featureToRef,
  pushNotification,
  useAuthStore,
  useCurrentUser,
  usePrincipalStore,
  useUserStore,
} from "@/store";
import { BBSwitch } from "@/bbkit";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { useRouter } from "vue-router";
import { UpdateUserRequest } from "@/types/proto/v1/auth_service";
import RegenerateRecoveryCodesView from "@/components/RegenerateRecoveryCodesView.vue";
import { useI18n } from "vue-i18n";

interface LocalState {
  editing: boolean;
  editingPrincipal?: PrincipalPatch;
  passwordConfirm?: string;
  showFeatureModal: boolean;
  showDisable2FAConfirmModal: boolean;
  showRegenerateRecoveryCodesView: boolean;
}

const props = defineProps<{
  principalId?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const authStore = useAuthStore();
const currentUser = useCurrentUser();
const userStore = useUserStore();
const principalStore = usePrincipalStore();
const state = reactive<LocalState>({
  editing: false,
  showFeatureModal: false,
  showDisable2FAConfirmModal: false,
  showRegenerateRecoveryCodesView: false,
});

const editNameTextField = ref();
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

const principal = computed(() => {
  if (props.principalId) {
    return principalStore.principalById(parseInt(props.principalId));
  }
  return currentUser.value;
});

const passwordMismatch = computed(() => {
  return (
    !isEmpty(state.editingPrincipal?.password) &&
    state.editingPrincipal?.password != state.passwordConfirm
  );
});

// User can change her own info.
// Besides, owner can also change anyone's info. This is for resetting password in case user forgets.
const allowEdit = computed(() => {
  return (
    currentUser.value.id === principal.value.id ||
    hasWorkspacePermission(
      "bb.permission.workspace.manage-member",
      currentUser.value.role
    )
  );
});

const allowSaveEdit = computed(() => {
  return (
    !isEqual(principal.value, state.editingPrincipal) &&
    (state.passwordConfirm === "" ||
      state.passwordConfirm === state.editingPrincipal?.password)
  );
});

const updatePrincipal = (field: string, value: string) => {
  (state.editingPrincipal as any)[field] = value;
};

const editUser = () => {
  const clone = cloneDeep(principal.value);
  state.editingPrincipal = {
    name: clone.name,
    email: clone.email,
    type: clone.type,
  };
  state.editing = true;

  nextTick(() => editNameTextField.value.focus());
};

const cancelEdit = () => {
  state.editingPrincipal = undefined;
  state.editing = false;
};

const saveEdit = async () => {
  await principalStore.patchPrincipal({
    principalId: principal.value.id,
    principalPatch: state.editingPrincipal!,
  });
  state.editingPrincipal = undefined;
  state.editing = false;
};

const handle2FAEnableStatusChanged = (enabled: boolean) => {
  if (!has2FAFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  if (enabled) {
    router.push({ name: "setting.profile.two-factor" });
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
      updateMask: ["user.mfa_enabled"],
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
