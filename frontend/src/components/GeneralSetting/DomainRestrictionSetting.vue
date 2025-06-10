<template>
  <div>
    <h3
      id="domain-restriction"
      class="font-medium flex flex-row justify-start items-center"
    >
      <span class="mr-2">{{
        $t("settings.general.workspace.domain-restriction.self")
      }}</span>
    </h3>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.domain-restriction.description") }}
    </p>
    <div class="w-full flex flex-col gap-2 mt-2">
      <NInput
        v-model:value="state.domain"
        :disabled="!allowEdit"
        :placeholder="
          $t(
            'settings.general.workspace.domain-restriction.domain-input-placeholder'
          )
        "
        type="text"
      />

      <div class="w-full flex flex-row justify-between items-center">
        <NCheckbox
          v-model:checked="state.enableRestriction"
          :disabled="!state.domain || !hasFeature || !allowEdit"
        >
          <div class="font-medium flex items-center gap-x-2">
            {{
              $t(
                "settings.general.workspace.domain-restriction.members-restriction.self"
              )
            }}
            <FeatureBadge :feature="PlanLimitConfig_Feature.USER_EMAIL_DOMAIN_RESTRICTION" />
          </div>
          <p class="text-sm text-gray-400 leading-tight">
            {{
              $t(
                "settings.general.workspace.domain-restriction.members-restriction.description"
              )
            }}
          </p>
        </NCheckbox>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head, isEqual } from "lodash-es";
import { NCheckbox, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import { FeatureBadge } from "../FeatureGuard";

const initialState = computed((): LocalState => {
  const defaultState: LocalState = {
    domain: "",
    enableRestriction: false,
  };
  if (
    Array.isArray(settingV1Store.workspaceProfileSetting?.domains) &&
    settingV1Store.workspaceProfileSetting?.domains.length > 0
  ) {
    defaultState.domain =
      head(settingV1Store.workspaceProfileSetting?.domains) || "";
    defaultState.enableRestriction =
      settingV1Store.workspaceProfileSetting?.enforceIdentityDomain || false;
  }
  return defaultState;
});

interface LocalState {
  domain: string;
  enableRestriction: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(initialState.value);

const hasFeature = featureToRef(PlanLimitConfig_Feature.USER_EMAIL_DOMAIN_RESTRICTION);

defineExpose({
  isDirty: computed(() => !isEqual(state, initialState.value)),
  update: async () => {
    if (state.domain.length === 0) {
      state.enableRestriction = false;
    }
    const updateMask: string[] = [];
    if (initialState.value.enableRestriction !== state.enableRestriction) {
      updateMask.push(
        "value.workspace_profile_setting_value.enforce_identity_domain"
      );
    }
    if (initialState.value.domain !== state.domain) {
      updateMask.push("value.workspace_profile_setting_value.domains");
    }
    if (updateMask.length > 0) {
      await settingV1Store.updateWorkspaceProfile({
        payload: {
          domains: state.domain ? [state.domain] : [],
          enforceIdentityDomain: state.enableRestriction,
        },
        updateMask,
      });
    }
  },
  revert: () => {
    Object.assign(state, initialState.value);
  },
});
</script>
