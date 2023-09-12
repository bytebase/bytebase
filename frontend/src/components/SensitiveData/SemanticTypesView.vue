<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click=""
      >
        {{ $t("settings.sensitive-data.semantic-types.add-type") }}
      </NButton>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <div
        v-if="state.semanticTypeList.length === 0"
        class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
      >
        <div class="text-center flex flex-col justify-center items-center">
          <img src="../../assets/illustration/no-data.webp" class="w-52" />
        </div>
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { NPopconfirm } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { Factor } from "@/plugins/cel";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingRulePolicy_MaskingRule,
} from "@/types/proto/v1/org_policy_service";
import { SemanticCategorySetting_SemanticCategory } from "@/types/proto/v1/setting_service";
import { arraySwap, hasWorkspacePermissionV1 } from "@/utils";
import {
  getClassificationLevelOptions,
  getEnvironmentIdOptions,
  getInstanceIdOptions,
  getProjectIdOptions,
} from "./components/utils";

interface LocalState {
  semanticTypeList: SemanticCategorySetting_SemanticCategory[];
  processing: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  semanticTypeList: [],
  processing: false,
});

const settingStore = useSettingV1Store();
const policyStore = usePolicyV1Store();
const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
</script>
