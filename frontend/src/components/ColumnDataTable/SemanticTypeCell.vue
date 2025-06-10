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
    :feature="PlanLimitConfig_Feature.DATA_MASKING"
    :instance="database.instanceResource"
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
import { PencilIcon, XIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed, reactive } from "vue";
import { useSemanticType } from "@/components/SensitiveData/useSemanticType";
import { MiniActionButton } from "@/components/v2";
import { useSubscriptionV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import SemanticTypesDrawer from "../SensitiveData/components/SemanticTypesDrawer.vue";

type LocalState = {
  showFeatureModal: boolean;
  showSemanticTypesDrawer: boolean;
};

const props = defineProps<{
  semanticTypeId: string;
  database: ComposedDatabase;
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
const { semanticType, semanticTypeList } = useSemanticType(
  computed(() => props.semanticTypeId)
);

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature(PlanLimitConfig_Feature.DATA_MASKING);
});

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    PlanLimitConfig_Feature.DATA_MASKING,
    props.database.instanceResource
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
