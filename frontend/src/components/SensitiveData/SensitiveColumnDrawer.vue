<template>
  <Drawer :show="true" @close="$emit('dismiss')">
    <DrawerContent
      :title="
        $t('settings.sensitive-data.column-detail.masking-setting-for-column', {
          column: mask.column,
        })
      "
    >
      <div class="divide-block-border divide-y space-y-8 w-[50rem] h-full">
        <div class="space-y-6">
          <div class="w-full">
            <h1 class="mb-2 font-semibold">
              {{ $t("settings.sensitive-data.masking-level.self") }}
            </h1>
            <MaskingLevelRadioGroup
              :level="state.maskingLevel"
              :level-list="MASKING_LEVELS"
              :disabled="!hasPermissionToUpdateConfig || state.processing"
              :effective-masking-level="columnMetadata?.effectiveMaskingLevel"
              @update:level="onMaskingLevelUpdate($event)"
            />
          </div>
          <div class="w-full">
            <div
              v-if="
                state.maskingLevel === MaskingLevel.FULL ||
                columnMetadata?.effectiveMaskingLevel === MaskingLevel.FULL
              "
              class="flex flex-col space-y-2"
            >
              <h1 class="font-semibold">
                {{ $t("settings.sensitive-data.algorithms.self") }}
              </h1>
              <span class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.semantic-types.table.full-masking-algorithm"
                  )
                }}
              </span>
              <NSelect
                :value="state.fullMaskingAlgorithmId"
                :options="algorithmList"
                :consistent-menu-width="false"
                :placeholder="columnDefaultMaskingAlgorithm"
                :fallback-option="
                  (_: string) => ({
                    label: columnDefaultMaskingAlgorithm,
                    value: '',
                  })
                "
                clearable
                size="small"
                style="min-width: 7rem; max-width: 20rem; overflow-x: hidden"
                @update:value="
                  (val) => {
                    state.partialMaskingAlgorithmId = val;
                    onMaskingAlgorithmChanged();
                  }
                "
              />
            </div>
            <div
              v-else-if="
                state.maskingLevel === MaskingLevel.PARTIAL ||
                columnMetadata?.effectiveMaskingLevel === MaskingLevel.PARTIAL
              "
              class="flex flex-col space-y-2"
            >
              <h1 class="font-semibold">
                {{ $t("settings.sensitive-data.algorithms.self") }}
              </h1>
              <span class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.semantic-types.table.partial-masking-algorithm"
                  )
                }}
              </span>
              <NSelect
                :value="state.partialMaskingAlgorithmId"
                :options="algorithmList"
                :consistent-menu-width="false"
                :placeholder="columnDefaultMaskingAlgorithm"
                :fallback-option="
                  (_: string) => ({
                    label: columnDefaultMaskingAlgorithm,
                    value: '',
                  })
                "
                clearable
                size="small"
                style="min-width: 7rem; max-width: 20rem; overflow-x: hidden"
                @update:value="
                  (val) => {
                    state.partialMaskingAlgorithmId = val;
                    onMaskingAlgorithmChanged();
                  }
                "
              />
            </div>
          </div>
        </div>
        <div class="pt-8 space-y-5">
          <div class="flex justify-between">
            <div>
              <h1 class="font-semibold">
                {{
                  $t("settings.sensitive-data.column-detail.access-user-list")
                }}
              </h1>
              <span class="textinfolabel">{{
                $t(
                  "settings.sensitive-data.column-detail.access-user-list-desc"
                )
              }}</span>
            </div>
            <NButton
              type="primary"
              :disabled="!hasPermissionToUpdatePolicy"
              @click="state.showGrantAccessDrawer = true"
            >
              {{ $t("settings.sensitive-data.grant-access") }}
            </NButton>
          </div>
          <MaskingExceptionUserTable
            size="small"
            :project="database.project"
            :disabled="state.processing"
            :show-database-column="false"
            :filter-exception="
              (exception) =>
                isCurrentColumnException(exception, {
                  maskData: mask,
                  database,
                })
            "
          />
        </div>
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>

  <GrantAccessDrawer
    v-if="state.showGrantAccessDrawer"
    :column-list="[
      {
        maskData: mask,
        database,
      },
    ]"
    :project-name="database.project"
    @dismiss="state.showGrantAccessDrawer = false"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NSelect, NButton } from "naive-ui";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { updateColumnConfig } from "@/components/ColumnDataTable/utils";
import type { MaskData } from "@/components/SensitiveData/types";
import { useSemanticType } from "@/components/SensitiveData/useSemanticType";
import { Drawer, DrawerContent } from "@/components/v2";
import { useSettingV1Store, useDBSchemaV1Store } from "@/store";
import { type ComposedDatabase } from "@/types";
import { MaskingLevel } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";
import GrantAccessDrawer from "./GrantAccessDrawer.vue";
import MaskingExceptionUserTable from "./MaskingExceptionUserTable.vue";
import MaskingLevelRadioGroup from "./components/MaskingLevelRadioGroup.vue";
import { isCurrentColumnException } from "./utils";

interface LocalState {
  processing: boolean;
  maskingLevel: MaskingLevel;
  showGrantAccessDrawer: boolean;
  fullMaskingAlgorithmId: string;
  partialMaskingAlgorithmId: string;
}

const props = defineProps<{
  mask: MaskData;
  database: ComposedDatabase;
}>();

defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  processing: false,
  maskingLevel: props.mask.maskingLevel,
  fullMaskingAlgorithmId: props.mask.fullMaskingAlgorithmId,
  partialMaskingAlgorithmId: props.mask.partialMaskingAlgorithmId,
  showGrantAccessDrawer: false,
});

const MASKING_LEVELS = [
  MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
  MaskingLevel.FULL,
  MaskingLevel.PARTIAL,
  MaskingLevel.NONE,
];

const { t } = useI18n();
const dbSchemaStore = useDBSchemaV1Store();
const settingStore = useSettingV1Store();
const { semanticType } = useSemanticType({
  database: props.database.name,
  schema: props.mask.schema,
  table: props.mask.table,
  column: props.mask.column,
});

const columnDefaultMaskingAlgorithm = computed(() => {
  if (semanticType.value) {
    return t("settings.sensitive-data.algorithms.default-with-semantic-type");
  }
  return t("settings.sensitive-data.algorithms.default");
});

const hasPermissionToUpdateConfig = computed(() => {
  return hasWorkspacePermissionV2("bb.databases.update");
});

const hasPermissionToUpdatePolicy = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

onMounted(() => {
  state.maskingLevel = props.mask.maskingLevel;
  state.fullMaskingAlgorithmId = props.mask.fullMaskingAlgorithmId;
  state.partialMaskingAlgorithmId = props.mask.partialMaskingAlgorithmId;
});

const onMaskingLevelUpdate = async (level: MaskingLevel) => {
  state.maskingLevel = level;
  await onColumnMaskingUpdate();

  dbSchemaStore.getOrFetchTableMetadata({
    database: props.database.name,
    schema: props.mask.schema,
    table: props.mask.table,
    skipCache: true,
    silent: false,
  });
};

const onMaskingAlgorithmChanged = async () => {
  await onColumnMaskingUpdate();
};

const onColumnMaskingUpdate = async () => {
  state.processing = true;

  try {
    await updateColumnConfig({
      database: props.database.name,
      schema: props.mask.schema,
      table: props.mask.table,
      column: props.mask.column,
      config: {
        maskingLevel: state.maskingLevel,
        fullMaskingAlgorithmId: state.fullMaskingAlgorithmId,
        partialMaskingAlgorithmId: state.partialMaskingAlgorithmId,
      },
    });
  } finally {
    state.processing = false;
  }
};

const columnMetadata = computedAsync(async () => {
  const { mask, database } = props;
  if (mask.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
    return undefined;
  }
  const table = await dbSchemaStore.getOrFetchTableMetadata({
    database: database.name,
    schema: mask.schema,
    table: mask.table,
  });
  return table?.columns.find((c) => c.name === mask.column);
}, undefined);

const algorithmList = computed((): SelectOption[] => {
  const list = (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));

  list.unshift({
    label: columnDefaultMaskingAlgorithm.value,
    value: "",
  });

  return list;
});
</script>
