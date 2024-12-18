<template>
  <div class="flex items-center">
    <span class="mr-2">
      {{ maskingLevelText }}
      <span v-if="maskingLevel === MaskingLevel.MASKING_LEVEL_UNSPECIFIED">
        ({{
          $t(
            `settings.sensitive-data.masking-level.${maskingLevelToJSON(
              column.effectiveMaskingLevel
            ).toLowerCase()}`
          )
        }})
      </span>
    </span>
    <NTooltip v-if="!isColumnConfigMasking">
      <template #trigger>
        <heroicons-outline:question-mark-circle class="shrink-0 h-4 w-4 mr-2" />
      </template>
      <i18n-t
        tag="div"
        keypath="settings.sensitive-data.column-detail.column-effective-masking-tips"
        class="whitespace-pre-line"
      >
        <template #link>
          <router-link
            class="flex items-center light-link text-sm"
            :to="{
              name: WORKSPACE_ROUTE_DATA_MASKING,
              hash: 'global-masking-rule',
            }"
          >
            {{ $t("settings.sensitive-data.global-rules.check-rules") }}
          </router-link>
        </template>
      </i18n-t>
    </NTooltip>
    <button
      v-if="!readonly"
      class="shrink-0 w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="openSensitiveDrawer()"
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

  <SensitiveColumnDrawer
    v-if="state.showSensitiveDataDrawer"
    :mask="{
      schema,
      table: table.name,
      column: column.name,
      maskingLevel,
      fullMaskingAlgorithmId: fullMaskingAlgorithm?.id ?? '',
      partialMaskingAlgorithmId: partialMaskingAlgorithm?.id ?? '',
    }"
    :database="database"
    @dismiss="state.showSensitiveDataDrawer = false"
  />
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useColumnMasking } from "@/components/SensitiveData/useColumnMasking";
import { WORKSPACE_ROUTE_DATA_MASKING } from "@/router/dashboard/workspaceRoutes";
import { useSubscriptionV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import type {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import SensitiveColumnDrawer from "../SensitiveData/SensitiveColumnDrawer.vue";

type LocalState = {
  showFeatureModal: boolean;
  showSensitiveDataDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showSensitiveDataDrawer: false,
});
const subscriptionV1Store = useSubscriptionV1Store();
const { maskingLevel, fullMaskingAlgorithm, partialMaskingAlgorithm } =
  useColumnMasking({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column: props.column.name,
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

const maskingLevelText = computed(() => {
  const level = maskingLevelToJSON(maskingLevel.value);
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
});

const isColumnConfigMasking = computed(() => {
  return maskingLevel.value !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
});

const openSensitiveDrawer = () => {
  if (!hasSensitiveDataFeature.value || instanceMissingLicense.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showSensitiveDataDrawer = true;
};
</script>
