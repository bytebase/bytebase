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
import { NButton, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { LocationQueryRaw } from "vue-router";
import { useRouter } from "vue-router";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import { getLocalSheetByName, getNextLocalSheetUID } from "@/components/Plan";
import { Drawer, DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  batchGetOrFetchDatabases,
  pushNotification,
  useCurrentProjectV1,
  useDatabaseV1Store,
  useDBGroupStore,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { DatabaseChangeType } from "@/types/proto-es/v1/common_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";

const props = defineProps<{
  title?: string;
  preSelectedDatabases?: ComposedDatabase[];
  preSelectedDatabaseGroup?: string;
  projectName?: string;
  useLegacyIssueFlow?: boolean;
}>();

const emit = defineEmits<{
  (event: "created", spec: Plan_Spec): void;
}>();

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const show = defineModel<boolean>("show", { default: false });

const isCreating = ref(false);

const databaseSelectState = reactive<DatabaseSelectState>({
  changeSource: "DATABASE",
  selectedDatabaseNameList: [],
});

const hasPreSelectedDatabases = computed(() => {
  return (
    (props.preSelectedDatabases?.length ?? 0) > 0 ||
    !!props.preSelectedDatabaseGroup
  );
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
      databaseSelectState.selectedDatabaseNameList = [];
      databaseSelectState.selectedDatabaseGroup = undefined;
    }

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

const handleConfirm = async () => {
  if (!canSubmit.value) return;

  isCreating.value = true;
  try {
    // Get targets
    const targets: string[] = [];
    if (props.preSelectedDatabaseGroup) {
      targets.push(props.preSelectedDatabaseGroup);
    } else if (props.preSelectedDatabases) {
      targets.push(...(props.preSelectedDatabases.map((db) => db.name) ?? []));
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
      const type = "bb.issue.database.schema.update";

      let databaseNames: string[] = [];
      // For database groups, use the group title for the issue title
      if (props.preSelectedDatabaseGroup) {
        const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(
          props.preSelectedDatabaseGroup
        );
        databaseNames = [dbGroup.title];
      } else if (props.preSelectedDatabases) {
        // Pre-selected individual databases
        databaseNames = props.preSelectedDatabases.map((db) => db.databaseName);
      } else {
        // Fetch database names from targets
        databaseNames = await Promise.all(
          targets.map(async (target) => {
            const db = await databaseStore.getOrFetchDatabaseByName(target);
            return db.databaseName;
          })
        );
      }

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
    localSheet.title = "Schema Migration";

    // Create spec
    const spec = createProto(Plan_SpecSchema, {
      id: uuidv4(),
      config: {
        case: "changeDatabaseConfig",
        value: createProto(Plan_ChangeDatabaseConfigSchema, {
          targets,
          type: DatabaseChangeType.MIGRATE,
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
