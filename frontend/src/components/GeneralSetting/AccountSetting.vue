<template>
  <div id="account" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>

    <div class="flex-1 lg:px-4">
      <div v-if="!isSaaSMode" class="mb-7 mt-4 lg:mt-0">
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="state.disallowSignup"
            :text="true"
            :disabled="!allowEdit || !hasDisallowSignupFeature"
          />
          <div class="font-medium flex items-center gap-x-2">
            {{ $t("settings.general.workspace.disallow-signup.enable") }}
            <FeatureBadge
              :feature="PlanFeature.FEATURE_DISALLOW_SELF_SERVICE_SIGNUP"
            />
          </div>
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.disallow-signup.description") }}
        </div>
      </div>

      <NDivider v-if="!isSaaSMode" />

      <PasswordRestrictionSetting
        ref="passwordRestrictionSettingRef"
        :allow-edit="allowEdit"
      />
      <NDivider />

      <div>
        <div class="mb-7 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2">
            <Switch
              v-model:value="state.require2fa"
              :text="true"
              :disabled="!allowEdit || !has2FAFeature"
            />
            <div class="font-medium flex items-center gap-x-2">
              {{ $t("settings.general.workspace.require-2fa.enable") }}
              <FeatureBadge :feature="PlanFeature.FEATURE_TWO_FA" />
            </div>
          </div>
          <div class="mt-1 mb-3 text-sm text-gray-400">
            {{ $t("settings.general.workspace.require-2fa.description") }}
          </div>
        </div>
        <div class="mb-7 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2">
            <Switch
              v-model:value="state.disallowPasswordSignin"
              :text="true"
              :disabled="
                !allowEdit ||
                !hasDisallowPasswordSigninFeature ||
                (!state.disallowPasswordSignin && !existActiveIdentityProvider)
              "
            />
            <div class="font-medium flex items-center gap-x-2">
              {{
                $t("settings.general.workspace.disallow-password-signin.enable")
              }}
              <NTooltip
                v-if="
                  allowEdit &&
                  hasDisallowPasswordSigninFeature &&
                  !state.disallowPasswordSignin &&
                  !existActiveIdentityProvider
                "
              >
                <template #trigger>
                  <TriangleAlertIcon class="w-4 text-warning" />
                </template>
                {{
                  $t(
                    "settings.general.workspace.disallow-password-signin.require-sso-setup"
                  )
                }}
              </NTooltip>
              <FeatureBadge
                :feature="PlanFeature.FEATURE_DISALLOW_PASSWORD_SIGNIN"
              />
            </div>
          </div>
          <div class="mt-1 mb-3 text-sm text-gray-400">
            {{
              $t(
                "settings.general.workspace.disallow-password-signin.description"
              )
            }}
          </div>
        </div>
      </div>
      <NDivider />

      <TokenDurationSetting
        ref="tokenDurationSettingRef"
        :allow-edit="allowEdit"
      />
    </div>
  </div>

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { TriangleAlertIcon } from "lucide-vue-next";
import { NDivider, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watchEffect } from "vue";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  useActuatorV1Store,
  useIdentityProviderStore,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { type WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import PasswordRestrictionSetting from "./PasswordRestrictionSetting.vue";
import TokenDurationSetting from "./TokenDurationSetting.vue";

interface LocalState {
  featureNameForModal?: PlanFeature;
  disallowSignup: boolean;
  require2fa: boolean;
  disallowPasswordSignin: boolean;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();
const idpStore = useIdentityProviderStore();
const passwordRestrictionSettingRef =
  ref<InstanceType<typeof PasswordRestrictionSetting>>();
const tokenDurationSettingRef =
  ref<InstanceType<typeof TokenDurationSetting>>();

const getInitialState = (): LocalState => {
  return {
    disallowSignup:
      settingV1Store.workspaceProfileSetting?.disallowSignup ?? false,
    require2fa: settingV1Store.workspaceProfileSetting?.require2fa ?? false,
    disallowPasswordSignin:
      settingV1Store.workspaceProfileSetting?.disallowPasswordSignin ?? false,
  };
};

const state = reactive<LocalState>({
  ...getInitialState(),
});

const { isSaaSMode } = storeToRefs(actuatorStore);
const has2FAFeature = featureToRef(PlanFeature.FEATURE_TWO_FA);
const hasDisallowSignupFeature = featureToRef(
  PlanFeature.FEATURE_DISALLOW_SELF_SERVICE_SIGNUP
);
const hasDisallowPasswordSigninFeature = featureToRef(
  PlanFeature.FEATURE_DISALLOW_PASSWORD_SIGNIN
);

watchEffect(async () => {
  await idpStore.fetchIdentityProviderList();
});

const existActiveIdentityProvider = computed(() => {
  return idpStore.identityProviderList.length > 0;
});

const isDirty = computed(() => {
  return (
    passwordRestrictionSettingRef.value?.isDirty ||
    tokenDurationSettingRef.value?.isDirty ||
    !isEqual(state, getInitialState())
  );
});

const onUpdate = async () => {
  if (passwordRestrictionSettingRef.value?.isDirty) {
    await passwordRestrictionSettingRef.value.update();
  }
  if (tokenDurationSettingRef.value?.isDirty) {
    await tokenDurationSettingRef.value.update();
  }

  const updateMaskPaths = [];
  const payload: Partial<WorkspaceProfileSetting> = {
    disallowSignup: state.disallowSignup,
    require2fa: state.require2fa,
    disallowPasswordSignin: state.disallowPasswordSignin,
  };
  if (
    state.disallowSignup !==
    settingV1Store.workspaceProfileSetting?.disallowSignup
  ) {
    updateMaskPaths.push("value.workspace_profile.disallow_signup");
  }
  if (state.require2fa !== settingV1Store.workspaceProfileSetting?.require2fa) {
    updateMaskPaths.push("value.workspace_profile.require_2fa");
  }
  if (
    state.disallowPasswordSignin !==
    settingV1Store.workspaceProfileSetting?.disallowPasswordSignin
  ) {
    updateMaskPaths.push("value.workspace_profile.disallow_password_signin");
  }

  if (updateMaskPaths.length > 0) {
    await settingV1Store.updateWorkspaceProfile({
      payload,
      updateMask: create(FieldMaskSchema, {
        paths: updateMaskPaths,
      }),
    });
  }
};

defineExpose({
  isDirty,
  title: props.title,
  update: onUpdate,
  revert: () => {
    Object.assign(state, getInitialState());
    passwordRestrictionSettingRef.value?.revert();
    tokenDurationSettingRef.value?.revert();
  },
});
</script>
