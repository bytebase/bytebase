<template>
  <div class="flex items-center">
    {{ semanticType?.title }}
    <button
      v-if="!readonly && semanticType"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="onSemanticTypeApply('')"
    >
      <heroicons-outline:x class="w-4 h-4" />
    </button>
    <button
      v-if="!readonly"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="openSemanticTypeDrawer()"
    >
      <heroicons-outline:pencil class="w-4 h-4" />
    </button>
  </div>

  <FeatureModal
    feature="bb.feature.sensitive-data"
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
import { computed } from "vue";
import { reactive } from "vue";
import { useSemanticType } from "@/components/SensitiveData/useSemanticType";
import { useSubscriptionV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import SemanticTypesDrawer from "../SensitiveData/components/SemanticTypesDrawer.vue";
import { updateColumnConfig } from "./utils";

type LocalState = {
  showFeatureModal: boolean;
  showSemanticTypesDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: string;
  column: string;
  readonly?: boolean;
}>();

const state = reactive<LocalState>({
  showFeatureModal: false,
  showSemanticTypesDrawer: false,
});
const subscriptionV1Store = useSubscriptionV1Store();
const { semanticType, semanticTypeList } = useSemanticType({
  database: props.database.name,
  schema: props.schema,
  table: props.table,
  column: props.column,
});

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    "bb.feature.sensitive-data",
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

const onSemanticTypeApply = async (semanticTypeId: string) => {
  await updateColumnConfig({
    database: props.database.name,
    schema: props.schema,
    table: props.table,
    column: props.column,
    columnCatalog: { semanticTypeId },
  });
};
</script>
