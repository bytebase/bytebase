<template>
  <div class="flex items-center">
    {{ columnSemanticType?.title }}
    <button
      v-if="!readonly && columnSemanticType"
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
import {
  useDBSchemaV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { updateColumnConfig } from "./utils";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";

type LocalState = {
  showFeatureModal: boolean;
  showSemanticTypesDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
}>();

const state = reactive<LocalState>({
  showFeatureModal: false,
  showSemanticTypesDrawer: false,
});
const subscriptionV1Store = useSubscriptionV1Store();
const settingV1Store = useSettingV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

const instanceMissingLicense = computed(() => {
  return subscriptionV1Store.instanceMissingLicense(
    "bb.feature.sensitive-data",
    props.database.instanceResource
  );
});

const semanticTypeList = computed(() => {
  return (
    settingV1Store.getSettingByName("bb.workspace.semantic-types")?.value
      ?.semanticTypeSettingValue?.types ?? []
  );
});

const columnSemanticType = computed(() => {
  const config = dbSchemaV1Store.getColumnConfig(
    props.database.name,
    props.schema,
    props.table.name,
    props.column.name
  );
  if (!config.semanticTypeId) {
    return;
  }
  return semanticTypeList.value.find(
    (data) => data.id === config.semanticTypeId
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
  await updateColumnConfig(
    props.database.name,
    props.schema,
    props.table.name,
    props.column.name,
    { semanticTypeId }
  );
};
</script>
