<template>
  <div class="flex items-center gap-x-1">
    <span v-if="semanticType?.title">{{ semanticType?.title }}</span>
    <span v-else class="text-control-placeholder italic"> N/A </span>
    <NPopconfirm
      v-if="!readonly && semanticType"
      @positive-click="() => onSemanticTypeApply('')"
    >
      <template #trigger>
        <MiniActionButton v-if="!readonly && semanticType">
          <XIcon class="w-3 h-3" />
        </MiniActionButton>
      </template>

      <template #default>
        <div>
          {{ $t("settings.sensitive-data.remove-semantic-type-tips") }}
        </div>
      </template>
    </NPopconfirm>
    <MiniActionButton v-if="!readonly" @click.prevent="openSemanticTypeDrawer">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_DATA_MASKING"
    :instance="getInstanceResource(database)"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <SemanticTypesDrawer
    v-if="state.showSemanticTypesDrawer"
    :show="true"
    :semantic-type-list="semanticTypeList"
    @dismiss="state.showSemanticTypesDrawer = false"
    @apply="onSemanticTypeApply($event)"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { PencilIcon, XIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store, useSubscriptionV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  SemanticTypeSetting_SemanticTypeSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { getInstanceResource, hasWorkspacePermissionV2 } from "@/utils";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import SemanticTypesDrawer from "./SemanticTypesDrawer.vue";

type LocalState = {
  showFeatureModal: boolean;
  showSemanticTypesDrawer: boolean;
};

const props = defineProps<{
  semanticTypeId: string;
  database: Database;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "apply", id: string): Promise<void>;
}>();

const state = reactive<LocalState>({
  showFeatureModal: false,
  showSemanticTypesDrawer: false,
});
const subscriptionV1Store = useSubscriptionV1Store();
const settingV1Store = useSettingV1Store();

watchEffect(async () => {
  await settingV1Store.getOrFetchSettingByName(
    Setting_SettingName.SEMANTIC_TYPES,
    true
  );
});

const semanticTypeList = computed(() => {
  const setting = settingV1Store.getSettingByName(
    Setting_SettingName.SEMANTIC_TYPES
  );
  if (setting?.value?.value?.case === "semanticType") {
    return setting.value.value.value.types ?? [];
  }
  return [];
});

const semanticType = computed(() => {
  const id = props.semanticTypeId;
  if (!id) return;
  if (!hasWorkspacePermissionV2("bb.settings.get")) {
    return create(SemanticTypeSetting_SemanticTypeSchema, {
      id,
      title: id,
    });
  }
  return semanticTypeList.value.find((data) => data.id === id);
});

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature(PlanFeature.FEATURE_DATA_MASKING);
});

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    PlanFeature.FEATURE_DATA_MASKING,
    getInstanceResource(props.database)
  );
});

const openSemanticTypeDrawer = () => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showSemanticTypesDrawer = true;
};

const onSemanticTypeApply = async (semanticType: string) => {
  await emit("apply", semanticType);
};
</script>
