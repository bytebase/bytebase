<template>
  <Drawer
    v-model:show="show"
    :mask-closable="false"
    placement="right"
    :resizable="true"
  >
    <DrawerContent
      :title="title ?? $t('plan.add-spec')"
      closable
      class="max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep">
          <NStep :title="changeTypeTitle" />
          <NStep :title="$t('plan.select-targets')" />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Change Type -->
          <template v-if="currentStep === Step.SELECT_CHANGE_TYPE">
            <NRadioGroup
              v-model:value="selectedMigrationType"
              size="large"
              class="gap-y-4 w-full md:w-[80vw] lg:w-[60vw]"
            >
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': isMigrateSelected,
                }"
              >
                <NRadio :value="MigrationType.DDL" class="w-full">
                  <div class="flex items-start gap-x-3 w-full">
                    <FileDiffIcon
                      class="w-6 h-6 mt-1 shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center gap-x-2">
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
                  'border-blue-500 bg-blue-50': isDMLSelected,
                }"
              >
                <NRadio :value="MigrationType.DML" class="w-full">
                  <div class="flex items-start gap-x-3 w-full">
                    <EditIcon
                      class="w-6 h-6 mt-1 shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center gap-x-2">
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
          <template v-else-if="currentStep === Step.SELECT_TARGETS">
            <div class="w-full md:w-[80vw] lg:w-[60vw]">
              <DatabaseAndGroupSelector
                :project="project"
                :value="databaseSelectState"
                @update:value="handleUpdateSelection"
              />
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="w-full flex items-center justify-end">
          <div class="flex items-center gap-x-3">
            <NButton
              quaternary
              v-if="currentStep === Step.SELECT_CHANGE_TYPE"
              @click="handleCancel"
            >
              {{ $t("common.close") }}
            </NButton>
            <NButton
              v-if="currentStep > Step.SELECT_CHANGE_TYPE"
              quaternary
              @click="handlePrevStep"
            >
              {{ $t("common.back") }}
            </NButton>
            <NButton
              v-if="!isLastStep"
              type="primary"
              :disabled="!canProceedToNextStep"
              @click="handleNextStep"
            >
              {{ $t("common.next") }}
            </NButton>
            <NButton
              v-else
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
import {
  useCurrentProjectV1,
  batchGetOrFetchDatabases,
  pushNotification,
} from "@/store";
import {
  DatabaseChangeType,
  MigrationType,
} from "@/types/proto-es/v1/common_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";

defineProps<{
  title?: string;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

enum Step {
  SELECT_CHANGE_TYPE = 1,
  SELECT_TARGETS = 2,
}

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const show = defineModel<boolean>("show", { default: false });

const selectedMigrationType: Ref<MigrationType> = ref(MigrationType.DDL);
const isCreating = ref(false);
const currentStep = ref(Step.SELECT_CHANGE_TYPE);

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
  return hasSelection.value && selectedMigrationType.value;
});

const canProceedToNextStep = computed(() => {
  if (currentStep.value === Step.SELECT_CHANGE_TYPE) {
    return !!selectedMigrationType.value;
  }
  if (currentStep.value === Step.SELECT_TARGETS) {
    return hasSelection.value;
  }
  return false;
});

const isLastStep = computed(() => {
  return currentStep.value === Step.SELECT_TARGETS;
});

const changeTypeTitle = computed(() => {
  if (currentStep.value !== Step.SELECT_CHANGE_TYPE) {
    if (selectedMigrationType.value === MigrationType.DDL) {
      return t("plan.schema-migration");
    } else if (selectedMigrationType.value === MigrationType.DML) {
      return t("plan.data-change");
    }
  }
  return t("plan.change-type");
});

const isMigrateSelected = computed(() => {
  return selectedMigrationType.value === MigrationType.DDL;
});

const isDMLSelected = computed(() => {
  return selectedMigrationType.value === MigrationType.DML;
});

// Reset state when drawer opens
watch(show, (newVal) => {
  if (newVal) {
    currentStep.value = Step.SELECT_CHANGE_TYPE;
    selectedMigrationType.value = MigrationType.DDL;
    databaseSelectState.changeSource = "DATABASE";
    databaseSelectState.selectedDatabaseNameList = [];
    databaseSelectState.selectedDatabaseGroup = undefined;
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
    await batchGetOrFetchDatabases(newState.selectedDatabaseNameList);
  }
};

const handleCancel = () => {
  show.value = false;
};

const handleNextStep = async () => {
  if (
    currentStep.value === Step.SELECT_CHANGE_TYPE &&
    selectedMigrationType.value
  ) {
    currentStep.value = Step.SELECT_TARGETS;
  }
};

const handlePrevStep = () => {
  if (currentStep.value === Step.SELECT_TARGETS) {
    currentStep.value = Step.SELECT_CHANGE_TYPE;
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
    const localSheet = getLocalSheetByName(
      `${project.value.name}/sheets/${sheetUID}`
    );
    localSheet.title =
      selectedMigrationType.value === MigrationType.DDL
        ? "Schema Migration"
        : "Data Change";

    // Create spec
    const spec = createProto(Plan_SpecSchema, {
      id: uuidv4(),
      config: {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,
          type: DatabaseChangeType.MIGRATE,
          migrationType: selectedMigrationType.value,
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
