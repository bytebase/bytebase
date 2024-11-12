<template>
  <Drawer :show="true" @close="$emit('dismiss')">
    <DrawerContent
      :title="
        $t('settings.sensitive-data.column-detail.masking-setting-for-column', {
          column: column.maskData.column,
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
              :disabled="!hasPermission || state.processing"
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
              :disabled="!hasPermission"
              @click="state.showGrantAccessDrawer = true"
            >
              {{ $t("settings.sensitive-data.grant-access") }}
            </NButton>
          </div>
          <MaskingExceptionUserTable
            size="small"
            :project="column.database.project"
            :disabled="state.processing"
            :show-database-column="false"
            :filter-exception="
              (exception) => isCurrentColumnException(exception, column)
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
    :column-list="[props.column]"
    :project-name="props.column.database.project"
    @dismiss="state.showGrantAccessDrawer = false"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NSelect, NButton } from "naive-ui";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useSemanticType } from "@/components/SensitiveData/useSemanticType";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  useSettingV1Store,
  usePolicyV1Store,
  pushNotification,
  useDBSchemaV1Store,
} from "@/store";
import { MaskingLevel } from "@/types/proto/v1/common";
import type { Policy, MaskData } from "@/types/proto/v1/org_policy_service";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import GrantAccessDrawer from "./GrantAccessDrawer.vue";
import MaskingExceptionUserTable from "./MaskingExceptionUserTable.vue";
import MaskingLevelRadioGroup from "./components/MaskingLevelRadioGroup.vue";
import type { SensitiveColumn } from "./types";
import { getMaskDataIdentifier, isCurrentColumnException } from "./utils";

interface LocalState {
  processing: boolean;
  maskingLevel: MaskingLevel;
  showGrantAccessDrawer: boolean;
  fullMaskingAlgorithmId: string;
  partialMaskingAlgorithmId: string;
}

const props = defineProps<{
  column: SensitiveColumn;
}>();

defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  processing: false,
  maskingLevel: props.column.maskData.maskingLevel,
  fullMaskingAlgorithmId: props.column.maskData.fullMaskingAlgorithmId,
  partialMaskingAlgorithmId: props.column.maskData.partialMaskingAlgorithmId,
  showGrantAccessDrawer: false,
});

const MASKING_LEVELS = [
  MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
  MaskingLevel.FULL,
  MaskingLevel.PARTIAL,
  MaskingLevel.NONE,
];

const { t } = useI18n();
const policyStore = usePolicyV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const settingStore = useSettingV1Store();
const { semanticType } = useSemanticType({
  database: props.column.database.name,
  schema: props.column.maskData.schema,
  table: props.column.maskData.table,
  column: props.column.maskData.column,
});

const columnDefaultMaskingAlgorithm = computed(() => {
  if (semanticType.value) {
    return t("settings.sensitive-data.algorithms.default-with-semantic-type");
  }
  return t("settings.sensitive-data.algorithms.default");
});

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

onMounted(() => {
  state.maskingLevel = props.column.maskData.maskingLevel;
  state.fullMaskingAlgorithmId = props.column.maskData.fullMaskingAlgorithmId;
  state.partialMaskingAlgorithmId =
    props.column.maskData.partialMaskingAlgorithmId;
});

const onMaskingLevelUpdate = async (level: MaskingLevel) => {
  state.maskingLevel = level;
  await onColumnMaskingUpdate();

  dbSchemaStore.getOrFetchTableMetadata({
    database: props.column.database.name,
    schema: props.column.maskData.schema,
    table: props.column.maskData.table,
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
    await upsertMaskingPolicy();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.processing = false;
  }
};

const upsertMaskingPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.column.database.name,
    policyType: PolicyType.MASKING,
  });

  const maskData = policy?.maskingPolicy?.maskData ?? [];
  const existedIndex = maskData.findIndex(
    (data) =>
      getMaskDataIdentifier(data) ===
      getMaskDataIdentifier(props.column.maskData)
  );
  const newData: MaskData = {
    ...props.column.maskData,
    maskingLevel: state.maskingLevel,
    fullMaskingAlgorithmId: state.fullMaskingAlgorithmId,
    partialMaskingAlgorithmId: state.partialMaskingAlgorithmId,
  };
  if (existedIndex < 0) {
    maskData.push(newData);
  } else {
    maskData[existedIndex] = newData;
  }

  const upsert: Partial<Policy> = {
    type: PolicyType.MASKING,
    resourceType: PolicyResourceType.DATABASE,
    maskingPolicy: {
      maskData,
    },
  };

  await policyStore.upsertPolicy({
    parentPath: props.column.database.name,
    policy: upsert,
  });
};

const columnMetadata = computedAsync(async () => {
  const { column } = props;
  if (column.maskData.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
    return undefined;
  }
  const table = await dbSchemaStore.getOrFetchTableMetadata({
    database: column.database.name,
    schema: column.maskData.schema,
    table: column.maskData.table,
  });
  return table?.columns.find((c) => c.name === column.maskData.column);
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
