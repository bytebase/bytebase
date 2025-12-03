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
      <NDynamicTags
        :size="'large'"
        :disabled="!allowEdit"
        :value="state.domains"
        :input-props="{
          placeholder: $t(
            'settings.general.workspace.domain-restriction.domain-input-placeholder'
          ),
          clearable: true,
        }"
        :input-style="'min-width: 20rem;'"
        @update:value="onDomainsUpdate"
      />

      <div class="w-full flex flex-row justify-between items-center">
        <NCheckbox
          v-model:checked="state.enableRestriction"
          :disabled="validDomains.length === 0 || !hasFeature || !allowEdit"
        >
          <div class="font-medium flex items-center gap-x-2">
            {{
              $t(
                "settings.general.workspace.domain-restriction.members-restriction.self"
              )
            }}
            <FeatureBadge
              :feature="PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION"
            />
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
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { NCheckbox, NDynamicTags } from "naive-ui";
import { computed, reactive } from "vue";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

const initialState = computed((): LocalState => {
  const defaultState: LocalState = {
    domains: [],
    enableRestriction: false,
  };
  if (Array.isArray(settingV1Store.workspaceProfileSetting?.domains)) {
    defaultState.domains = [...settingV1Store.workspaceProfileSetting?.domains];
    defaultState.enableRestriction =
      settingV1Store.workspaceProfileSetting?.enforceIdentityDomain || false;
  }
  return defaultState;
});

interface LocalState {
  domains: string[];
  enableRestriction: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(cloneDeep(initialState.value));

const hasFeature = featureToRef(
  PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION
);

const onDomainsUpdate = (values: string[]) => {
  state.domains = values;
  if (validDomains.value.length === 0) {
    state.enableRestriction = false;
  }
};

const validDomains = computed(() => {
  return state.domains.filter((domain) => !!domain);
});

defineExpose({
  isDirty: computed(
    () =>
      !isEqual(
        {
          ...state,
          domains: validDomains.value,
        },
        initialState.value
      )
  ),
  update: async () => {
    if (validDomains.value.length === 0) {
      state.enableRestriction = false;
    }
    const updateMask: string[] = [];
    if (initialState.value.enableRestriction !== state.enableRestriction) {
      updateMask.push("value.workspace_profile.enforce_identity_domain");
    }

    if (!isEqual(validDomains.value, initialState.value.domains)) {
      updateMask.push("value.workspace_profile.domains");
    }
    if (updateMask.length > 0) {
      await settingV1Store.updateWorkspaceProfile({
        payload: {
          domains: validDomains.value,
          enforceIdentityDomain: state.enableRestriction,
        },
        updateMask: create(FieldMaskSchema, { paths: updateMask }),
      });
    }
  },
  revert: () => {
    Object.assign(state, initialState.value);
  },
});
</script>
