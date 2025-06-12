<template>
  <Drawer
    v-model:show="show"
    :width="640"
    :mask-closable="true"
    placement="right"
    class="!w-[100vw] sm:!w-[80vw]"
  >
    <DrawerContent :title="title ?? $t('plan.add-spec')" closable>
      <div class="flex flex-col gap-y-4">
        <!-- Step 1: Select Change Type -->
        <div class="flex flex-row items-center gap-x-4">
          <label class="text-sm font-medium text-main block">
            {{ $t("plan.change-type") }}
          </label>
          <NRadioGroup v-model:value="changeType" class="space-y-2">
            <NRadio value="MIGRATE">
              <div class="flex items-center gap-2">
                <FileDiffIcon class="w-4 h-4" />
                <span>{{ $t("plan.schema-migration") }}</span>
              </div>
              <div class="text-sm text-control-light ml-6">
                {{ $t("plan.schema-migration-description") }}
              </div>
            </NRadio>
            <NRadio value="DATA">
              <div class="flex items-center gap-2">
                <EditIcon class="w-4 h-4" />
                <span>{{ $t("plan.data-change") }}</span>
              </div>
              <div class="text-sm text-control-light ml-6">
                {{ $t("plan.data-change-description") }}
              </div>
            </NRadio>
          </NRadioGroup>
        </div>

        <!-- Step 2: Select Targets -->
        <div>
          <label class="text-sm font-medium text-main mb-2 block">
            {{ $t("plan.select-targets") }}
          </label>
          <DatabaseAndGroupSelector
            :project="project"
            :value="databaseSelectState"
            @update:value="handleUpdateSelection"
          />
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="handleCancel">
            {{ $t("common.cancel") }}
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
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { FileDiffIcon, EditIcon } from "lucide-vue-next";
import { NButton, NRadio, NRadioGroup } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watch } from "vue";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
} from "@/types/proto/v1/plan_service";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { Sheet } from "@/types/proto/v1/sheet_service";

defineProps<{
  title?: string;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

const { project } = useCurrentProjectV1();
const show = defineModel<boolean>("show", { default: false });

const changeType = ref<"MIGRATE" | "DATA">("MIGRATE");
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
  return hasSelection.value && changeType.value;
});

// Reset state when drawer opens
watch(show, (newVal) => {
  if (newVal) {
    changeType.value = "MIGRATE";
    databaseSelectState.changeSource = "DATABASE";
    databaseSelectState.selectedDatabaseNameList = [];
    databaseSelectState.selectedDatabaseGroup = undefined;
    isCreating.value = false;
  }
});

const handleUpdateSelection = (newState: DatabaseSelectState) => {
  Object.assign(databaseSelectState, newState);
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

    const sheetUID = getNextLocalSheetUID();
    const sheet = Sheet.fromPartial({
      ...getLocalSheetByName(`${project.value.name}/sheets/${sheetUID}`),
      title:
        changeType.value === "MIGRATE" ? "Schema Migration" : "Data Change",
    });

    // Create spec
    const spec: Plan_Spec = {
      id: uuidv4(),
      changeDatabaseConfig: Plan_ChangeDatabaseConfig.fromPartial({
        targets,
        type:
          changeType.value === "MIGRATE"
            ? Plan_ChangeDatabaseConfig_Type.MIGRATE
            : Plan_ChangeDatabaseConfig_Type.DATA,
        sheet: sheet.name,
      }),
    };

    emit("created", spec);
    show.value = false;
  } finally {
    isCreating.value = false;
  }
};
</script>
