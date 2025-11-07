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
        <!-- Steps indicator (hide when databases are pre-selected) -->
        <NSteps v-if="!hasPreSelectedDatabases" :current="currentStep">
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
              class="gap-y-4 w-full flex! flex-col md:w-[80vw] lg:w-[60vw]"
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
          <div class="flex items-center gap-x-2">
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
import {
  NButton,
  NRadio,
  NRadioGroup,
  NSteps,
  NStep,
  useDialog,
} from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { Ref } from "vue";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { LocationQueryRaw } from "vue-router";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import { Drawer, DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  batchGetOrFetchDatabases,
  pushNotification,
  useDatabaseV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  DatabaseChangeType,
  MigrationType,
} from "@/types/proto-es/v1/common_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";

const props = defineProps<{
  title?: string;
  preSelectedDatabases?: ComposedDatabase[];
  projectName?: string;
  useLegacyIssueFlow?: boolean;
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
const router = useRouter();
const dialog = useDialog();
const databaseStore = useDatabaseV1Store();
const show = defineModel<boolean>("show", { default: false });

const selectedMigrationType: Ref<MigrationType> = ref(MigrationType.DDL);
const isCreating = ref(false);
const currentStep = ref(Step.SELECT_CHANGE_TYPE);

const databaseSelectState = reactive<DatabaseSelectState>({
  changeSource: "DATABASE",
  selectedDatabaseNameList: [],
});

const hasPreSelectedDatabases = computed(() => {
  return (props.preSelectedDatabases?.length ?? 0) > 0;
});

const hasSelection = computed(() => {
  if (hasPreSelectedDatabases.value) {
    return true;
  }
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
  if (hasPreSelectedDatabases.value) {
    return currentStep.value === Step.SELECT_CHANGE_TYPE;
  }
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

const showDatabaseDriftedWarningDialog = () => {
  return new Promise((resolve) => {
    dialog.create({
      type: "warning",
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.schema-drift-detected.self"),
      content: t("issue.schema-drift-detected.description"),
      autoFocus: false,
      onNegativeClick: () => {
        resolve(false);
      },
      onPositiveClick: () => {
        resolve(true);
      },
    });
  });
};

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
    if (hasPreSelectedDatabases.value) {
      targets.push(...(props.preSelectedDatabases?.map((db) => db.name) ?? []));
    } else if (databaseSelectState.changeSource === "DATABASE") {
      targets.push(...databaseSelectState.selectedDatabaseNameList);
    } else if (databaseSelectState.selectedDatabaseGroup) {
      targets.push(databaseSelectState.selectedDatabaseGroup);
    }

    // Check for database drift if using legacy issue flow
    if (props.useLegacyIssueFlow && hasPreSelectedDatabases.value) {
      if (props.preSelectedDatabases?.some((d) => d.drifted)) {
        const confirmed = await showDatabaseDriftedWarningDialog();
        if (!confirmed) {
          isCreating.value = false;
          return;
        }
      }
    }

    // If using legacy issue flow (from database dashboard), navigate to issue creation
    if (props.useLegacyIssueFlow) {
      const type =
        selectedMigrationType.value === MigrationType.DDL
          ? "bb.issue.database.schema.update"
          : "bb.issue.database.data.update";

      const databaseNames = hasPreSelectedDatabases.value
        ? (props.preSelectedDatabases?.map((db) => db.databaseName) ?? [])
        : await Promise.all(
            targets.map(async (target) => {
              const db = await databaseStore.getOrFetchDatabaseByName(target);
              return db.databaseName;
            })
          );

      const query: LocationQueryRaw = {
        template: type,
        name: generateIssueTitle(type, databaseNames),
        databaseList: targets.join(","),
      };

      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(
            props.projectName ?? project.value.name
          ),
          issueSlug: "create",
        },
        query,
      });

      show.value = false;
      isCreating.value = false;
      return;
    }

    // Otherwise, create spec for plan flow
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
