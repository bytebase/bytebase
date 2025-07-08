<template>
  <Drawer
    v-model:show="show"
    :width="640"
    :mask-closable="true"
    placement="right"
    class="!w-[1024px] !max-w-[100vw]"
  >
    <DrawerContent :title="title ?? $t('plan.add-spec')" closable>
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep">
          <NStep :title="stepTitle" />
          <NStep :title="$t('plan.select-targets')" />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Change Type -->
          <template v-if="currentStep === 1">
            <NRadioGroup
              v-model:value="selectedChangeType"
              size="large"
              class="space-y-4 w-full"
            >
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': isMigrateSelected,
                }"
              >
                <NRadio
                  :value="Plan_ChangeDatabaseConfig_Type.MIGRATE"
                  class="w-full"
                >
                  <div class="flex items-start space-x-3 w-full">
                    <FileDiffIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          <span>{{ $t("plan.schema-migration") }}</span>
                        </span>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{ $t("plan.schema-migration-description") }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': isDataSelected,
                }"
              >
                <NRadio
                  :value="Plan_ChangeDatabaseConfig_Type.DATA"
                  class="w-full"
                >
                  <div class="flex items-start space-x-3 w-full">
                    <EditIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          <span>{{ $t("plan.data-change") }}</span>
                        </span>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{ $t("plan.data-change-description") }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
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
              :disabled="!selectedChangeType"
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
import { create as createProto } from "@bufbuild/protobuf";
import { FileDiffIcon, EditIcon } from "lucide-vue-next";
import { NButton, NRadio, NRadioGroup, NSteps, NStep } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { Ref } from "vue";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_ChangeDatabaseConfigSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_SpecSchema,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

defineProps<{
  title?: string;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const show = defineModel<boolean>("show", { default: false });

const selectedChangeType: Ref<Plan_ChangeDatabaseConfig_Type> = ref(
  Plan_ChangeDatabaseConfig_Type.MIGRATE
);
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
  return hasSelection.value && selectedChangeType.value;
});

const stepTitle = computed(() => {
  if (currentStep.value === 1) {
    return t("plan.change-type");
  }
  return selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.DATA
    ? t("plan.data-change")
    : t("plan.schema-migration");
});

const isMigrateSelected = computed(() => {
  return selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.MIGRATE;
});

const isDataSelected = computed(() => {
  return selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.DATA;
});

// Reset state when drawer opens
watch(show, (newVal) => {
  if (newVal) {
    currentStep.value = 1;
    selectedChangeType.value = Plan_ChangeDatabaseConfig_Type.MIGRATE;
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
  if (currentStep.value === 1 && selectedChangeType.value) {
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
    const sheet = createProto(SheetSchema, {
      ...getLocalSheetByName(`${project.value.name}/sheets/${sheetUID}`),
      title:
        selectedChangeType.value === Plan_ChangeDatabaseConfig_Type.MIGRATE
          ? "Schema Migration"
          : "Data Change",
    });

    // Create spec
    const spec = createProto(Plan_SpecSchema, {
      id: uuidv4(),
      config: {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,
          type: selectedChangeType.value,
          sheet: sheet.name,
        }),
      },
    });

    emit("created", spec);
    show.value = false;
  } finally {
    isCreating.value = false;
  }
};
</script>
