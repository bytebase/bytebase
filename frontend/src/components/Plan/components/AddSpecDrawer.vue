<template>
  <Drawer
    placement="right"
    v-model:show="show"
    :mask-closable="true"
    :resizable="true"
  >
    <DrawerContent
      :title="title ?? $t('plan.add-spec')"
      closable
      class="max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-4">
        <!-- Select Targets -->
        <div class="w-full md:w-[80vw] lg:w-[60vw]">
          <DatabaseAndGroupSelector
            :project="project"
            :value="databaseSelectState"
            @update:value="handleUpdateSelection"
          />
        </div>

      </div>
      <template #footer>
        <div class="w-full flex items-center justify-end">
          <div class="flex items-center gap-x-2">
            <NButton quaternary @click="handleCancel">
              {{ $t("common.close") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!canSubmit"
              :loading="isCreating"
              @click="handleConfirm"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useCurrentProjectV1,
  useDatabaseV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";

const props = defineProps<{
  title?: string;
  preSelectedDatabases?: ComposedDatabase[];
  preSelectedDatabaseGroup?: string;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const dbStore = useDatabaseV1Store();
const show = defineModel<boolean>("show", { default: false });

const isCreating = ref(false);

const databaseSelectState = reactive<DatabaseSelectState>({
  changeSource: "DATABASE",
  selectedDatabaseNameList: [],
});

const hasSelection = computed(() => {
  if (databaseSelectState.changeSource === "DATABASE") {
    return databaseSelectState.selectedDatabaseNameList.length > 0;
  } else {
    return !!databaseSelectState.selectedDatabaseGroup;
  }
});

const canSubmit = computed(() => {
  return hasSelection.value;
});

// Reset state when drawer opens
watch(show, (newVal) => {
  if (newVal) {
    // Initialize with pre-selected database group if provided
    if (props.preSelectedDatabaseGroup) {
      databaseSelectState.changeSource = "GROUP";
      databaseSelectState.selectedDatabaseNameList = [];
      databaseSelectState.selectedDatabaseGroup =
        props.preSelectedDatabaseGroup;
    } else {
      databaseSelectState.changeSource = "DATABASE";
      databaseSelectState.selectedDatabaseNameList = (
        props.preSelectedDatabases ?? []
      ).map((db) => db.name);
      databaseSelectState.selectedDatabaseGroup = undefined;
    }

    isCreating.value = false;
  }
});

const handleUpdateSelection = async (newState: DatabaseSelectState) => {
  Object.assign(databaseSelectState, newState);

  // Preload database information if databases are selected
  if (
    newState.changeSource === "DATABASE" &&
    newState.selectedDatabaseNameList?.length > 0
  ) {
    await dbStore.batchGetOrFetchDatabases(newState.selectedDatabaseNameList);
  }
};

const handleCancel = () => {
  show.value = false;
};

const handleConfirm = async () => {
  if (!canSubmit.value) return;

  isCreating.value = true;
  try {
    // Get targets
    const targets: string[] = [];
    if (databaseSelectState.changeSource === "DATABASE") {
      targets.push(...databaseSelectState.selectedDatabaseNameList);
    } else if (databaseSelectState.selectedDatabaseGroup) {
      targets.push(databaseSelectState.selectedDatabaseGroup);
    }

    // Otherwise, create spec for plan flow
    const sheetUID = getNextLocalSheetUID();
    const localSheet = getLocalSheetByName(
      `${project.value.name}/sheets/${sheetUID}`
    );

    // Create spec
    const spec = createProto(Plan_SpecSchema, {
      id: uuidv4(),
      config: {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,

          sheet: localSheet.name,
        }),
      },
    });

    emit("created", spec);
    show.value = false;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: error instanceof Error ? error.message : String(error),
    });
  } finally {
    isCreating.value = false;
  }
};
</script>
