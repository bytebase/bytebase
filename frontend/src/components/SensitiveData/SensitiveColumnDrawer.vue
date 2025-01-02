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
              @update:level="onMaskingLevelUpdate($event)"
            />
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
import { NButton } from "naive-ui";
import { computed, reactive, onMounted } from "vue";
import { updateColumnConfig } from "@/components/ColumnDataTable/utils";
import type { MaskData } from "@/components/SensitiveData/types";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDBSchemaV1Store } from "@/store";
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
}

const props = defineProps<{
  mask: MaskData;
  database: ComposedDatabase;
}>();

defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  processing: false,
  maskingLevel: props.mask.maskingLevel,
  showGrantAccessDrawer: false,
});

const MASKING_LEVELS = [
  MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
  MaskingLevel.NONE,
];

const dbSchemaStore = useDBSchemaV1Store();

const hasPermissionToUpdateConfig = computed(() => {
  return hasWorkspacePermissionV2("bb.databases.update");
});

const hasPermissionToUpdatePolicy = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

onMounted(() => {
  state.maskingLevel = props.mask.maskingLevel;
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

const onColumnMaskingUpdate = async () => {
  state.processing = true;

  try {
    await updateColumnConfig({
      database: props.database.name,
      schema: props.mask.schema,
      table: props.mask.table,
      column: props.mask.column,
      columnCatalog: {
        maskingLevel: state.maskingLevel,
      },
    });
  } finally {
    state.processing = false;
  }
};
</script>
