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
        <!-- Steps indicator -->
        <NSteps :current="currentStep" size="small">
          <NStep
            :title="
              currentStep === 1
                ? $t('plan.change-type')
                : changeType === 'DATA'
                  ? $t('plan.data-change')
                  : $t('plan.schema-migration')
            "
          />
          <NStep :title="$t('plan.select-targets')" />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Change Type -->
          <template v-if="currentStep === 1">
            <NRadioGroup
              v-model:value="changeType"
              size="large"
              class="space-x-3"
            >
              <NRadio value="MIGRATE">
                <div class="flex items-center gap-2">
                  <FileDiffIcon class="w-5 h-5" />
                  <span>{{ $t("plan.schema-migration") }}</span>
                </div>
                <div class="text-sm text-control-light ml-7">
                  {{ $t("plan.schema-migration-description") }}
                </div>
              </NRadio>
              <NRadio value="DATA">
                <div class="flex items-center gap-2">
                  <EditIcon class="w-5 h-5" />
                  <span>{{ $t("plan.data-change") }}</span>
                </div>
                <div class="text-sm text-control-light ml-7">
                  {{ $t("plan.data-change-description") }}
                </div>
              </NRadio>
            </NRadioGroup>
          </template>

          <!-- Step 2: Select Targets -->
          <template v-else-if="currentStep === 2">
            <DatabaseAndGroupSelector
              :project="project"
              :value="databaseSelectState"
              @update:value="handleUpdateSelection"
            />
          </template>
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-x-3">
            <NButton v-if="currentStep === 1" @click="handleCancel">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              v-if="currentStep === 2"
              quaternary
              @click="handlePrevStep"
            >
              {{ $t("common.back") }}
            </NButton>
            <NButton
              v-if="currentStep === 1"
              type="primary"
              :disabled="!changeType"
              @click="handleNextStep"
            >
              {{ $t("common.next") }}
            </NButton>
            <NButton
              v-else-if="currentStep === 2"
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
import { FileDiffIcon, EditIcon } from "lucide-vue-next";
import { NButton, NRadio, NRadioGroup, NSteps, NStep } from "naive-ui";
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
const currentStep = ref(1);

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
    currentStep.value = 1;
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

const handleNextStep = () => {
  if (currentStep.value === 1 && changeType.value) {
    currentStep.value = 2;
  }
};

const handlePrevStep = () => {
  if (currentStep.value === 2) {
    currentStep.value = 1;
  }
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
