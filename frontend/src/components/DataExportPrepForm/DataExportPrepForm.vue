<template>
  <DrawerContent class="max-w-[100vw]">
    <template #header>
      <div class="flex flex-col gap-y-3">
        <span>
          {{ $t("custom-approval.risk-rule.risk.namespace.data_export") }}
        </span>
        <NSteps :current="state.step" size="small">
          <NStep :title="$t('plan.targets.title')" />
          <NStep :title="$t('common.configure')" />
        </NSteps>
      </div>
    </template>

    <div
      class="flex flex-col gap-y-4 h-full w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <template v-if="state.step === 1">
        <DatabaseAndGroupSelector
          :project="project"
          v-model:value="state.targetSelectState"
        />
      </template>

      <template v-else>
        <div class="flex flex-col gap-y-4 pb-2">
          <div class="flex flex-col gap-y-2">
            <h3 class="text-base font-medium">
              {{ $t("plan.targets.title") }}
            </h3>
            <div class="flex flex-wrap gap-2">
              <div
                v-for="target in targets"
                :key="target"
                class="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0"
              >
                <template v-if="isValidDatabaseName(target)">
                  <DatabaseDisplay
                    :database="target"
                    :show-environment="true"
                    size="medium"
                    class="flex-1 min-w-0"
                  />
                </template>
                <template v-else-if="isValidDatabaseGroupName(target)">
                  <DatabaseGroupTargetDisplay :target="target" class="px-1 py-1" />
                </template>
              </div>
            </div>
          </div>

          <div class="flex flex-col gap-y-2">
            <label class="textlabel">
              {{ $t("common.title") }}
              <RequiredStar v-if="project.enforceIssueTitle" />
            </label>
            <NInput
              v-model:value="state.title"
              :placeholder="$t('common.title')"
              @update:value="state.titleEdited = true"
            />
          </div>

          <div class="flex flex-col gap-y-2">
            <label class="textlabel">
              {{ $t("common.description") }}
            </label>
            <MarkdownEditor
              mode="editor"
              :content="state.description"
              :project="project"
              :compact="true"
              @change="state.description = $event"
            />
          </div>

          <div class="flex flex-col gap-y-2">
            <label class="textlabel">
              {{ $t("issue.labels") }}
              <RequiredStar v-if="project.forceIssueLabels" />
            </label>
            <IssueLabelSelector
              :selected="state.labels"
              :project="project"
              size="medium"
              :render-menu-inside-parent="true"
              @update:selected="state.labels = $event"
            />
          </div>

          <div class="flex flex-col gap-y-2">
            <label class="textlabel">
              {{ $t("common.sql") }}
              <RequiredStar />
            </label>
            <div class="h-96 overflow-hidden border rounded-sm">
              <MonacoEditor
                class="w-full h-full"
                :content="state.statement"
                language="sql"
                @update:content="state.statement = $event"
              />
            </div>
          </div>

          <div class="flex flex-col gap-y-2">
            <h3 class="text-base">
              {{ $t("issue.data-export.options") }}
            </h3>
            <div class="p-3 border rounded-sm flex flex-col gap-y-3">
              <div class="flex items-center gap-4">
                <span class="text-sm">
                  {{ $t("issue.data-export.format") }}
                </span>
                <ExportFormatSelector
                  v-model:format="state.format"
                  :editable="true"
                />
              </div>

              <ExportPasswordInputer
                v-model:password="state.password"
                :editable="true"
              />
            </div>
          </div>

          <LimitsSection />
        </div>
      </template>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-x-3">
        <NButton @click.prevent="handleCancel">
          {{
            state.step === 1 ? $t("common.cancel") : $t("common.back")
          }}
        </NButton>
        <NButton
          v-if="state.step === 1"
          type="primary"
          :disabled="!validSelectState"
          @click="state.step = 2"
        >
          {{ $t("common.next") }}
        </NButton>
        <NButton
          v-else
          type="primary"
          :disabled="!canCreate"
          :loading="state.creating"
          @click="handleCreate"
        >
          {{ $t("common.create") }}
        </NButton>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NInput, NStep, NSteps } from "naive-ui";
import { computed, reactive, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector/";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { createEmptyLocalSheet } from "@/components/Plan/logic";
import DatabaseGroupTargetDisplay from "@/components/Plan/components/SpecDetailView/DatabaseGroupTargetDisplay.vue";
import ExportFormatSelector from "@/components/Plan/components/ExportOption/ExportFormatSelector.vue";
import ExportPasswordInputer from "@/components/Plan/components/ExportOption/ExportPasswordInputer.vue";
import LimitsSection from "@/components/Plan/components/IssueReviewView/DatabaseExportView/LimitsSection.vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import MarkdownEditor from "@/components/MarkdownEditor";
import { DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  experimentalCreateIssueByPlan,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectByName,
  useSheetV1Store,
} from "@/store";
import {
  isValidDatabaseGroupName,
  isValidDatabaseName,
} from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { Issue_Type, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ExportDataConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractIssueUID,
  extractProjectResourceName,
  generatePlanTitle,
  setSheetStatement,
} from "@/utils";
import {
  normalizeDataExportPrepSeed,
  type DataExportPrepSeed,
} from "./types";

type LocalState = {
  step: 1 | 2;
  creating: boolean;
  targetSelectState?: DataExportPrepSeed["targetSelectState"];
  title: string;
  titleEdited: boolean;
  description: string;
  labels: string[];
  statement: string;
  format: ExportFormat;
  password: string;
};

const props = defineProps<{
  projectName: string;
  show?: boolean;
  seed?: DataExportPrepSeed;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUserV1();
const sheetStore = useSheetV1Store();
const dbStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const { project } = useProjectByName(computed(() => props.projectName));

const buildInitialState = (): LocalState => {
  const seed = normalizeDataExportPrepSeed(props.seed);
  return {
    step: seed.step,
    creating: false,
    targetSelectState: seed.targetSelectState,
    title: "",
    titleEdited: false,
    description: "",
    labels: [],
    statement: "",
    format: ExportFormat.JSON,
    password: "",
  };
};

const state = reactive<LocalState>(buildInitialState());

const resetState = () => {
  Object.assign(state, buildInitialState());
};

const targets = computed(() => {
  if (!state.targetSelectState) return [];
  if (state.targetSelectState.changeSource === "DATABASE") {
    return state.targetSelectState.selectedDatabaseNameList;
  }
  return state.targetSelectState.selectedDatabaseGroup
    ? [state.targetSelectState.selectedDatabaseGroup]
    : [];
});

const validSelectState = computed(() => targets.value.length > 0);

const targetTitleNames = computed(() => {
  return targets.value.map((target) => {
    if (isValidDatabaseName(target)) {
      return extractDatabaseResourceName(target).databaseName;
    }
    if (isValidDatabaseGroupName(target)) {
      return extractDatabaseGroupName(target);
    }
    return target;
  });
});

const canCreate = computed(() => {
  if (!validSelectState.value) return false;
  if (project.value.enforceIssueTitle && !state.title.trim()) return false;
  if (project.value.forceIssueLabels && state.labels.length === 0) return false;
  if (!state.statement.trim()) return false;
  return true;
});

const effectiveTitle = computed(() => {
  const title = state.title.trim();
  if (title) {
    return title;
  }
  return generatePlanTitle("bb.plan.export-data", targetTitleNames.value);
});

watch(
  () => targetTitleNames.value.join(","),
  (signature) => {
    if (!signature) return;
    if (state.titleEdited && state.title.trim()) return;
    state.title = generatePlanTitle("bb.plan.export-data", targetTitleNames.value);
  },
  { immediate: true }
);

watch(
  () => props.show,
  (show) => {
    if (show) {
      resetState();
    }
  },
  { immediate: true }
);

watchEffect(() => {
  for (const target of targets.value) {
    if (isValidDatabaseName(target)) {
      dbStore.getOrFetchDatabaseByName(target);
    } else if (isValidDatabaseGroupName(target)) {
      dbGroupStore.getOrFetchDBGroupByName(target, {
        view: DatabaseGroupView.FULL,
      });
    }
  }
});

const handleCancel = () => {
  if (state.step === 2) {
    state.step = 1;
    return;
  }
  resetState();
  emit("dismiss");
};

const handleCreate = async () => {
  if (state.creating || !canCreate.value) {
    return;
  }

  state.creating = true;

  try {
    const sheet = createEmptyLocalSheet();
    setSheetStatement(sheet, state.statement);
    const createdSheet = await sheetStore.createSheet(project.value.name, sheet);

    const spec = create(Plan_SpecSchema, {
      id: crypto.randomUUID(),
      config: {
        case: "exportDataConfig",
        value: create(Plan_ExportDataConfigSchema, {
          targets: targets.value,
          sheet: createdSheet.name,
          format: state.format,
          password: state.password,
        }),
      },
    });

    const planCreate = create(PlanSchema, {
      title: effectiveTitle.value,
      description: state.description,
      specs: [spec],
      creator: currentUser.value.name,
    });

    const issueCreate = create(IssueSchema, {
      title: effectiveTitle.value,
      description: state.description,
      creator: `users/${currentUser.value.email}`,
      labels: state.labels,
      type: Issue_Type.DATABASE_EXPORT,
    });

    const { createdIssue } = await experimentalCreateIssueByPlan(
      project.value,
      issueCreate,
      planCreate,
      { skipRollout: true }
    );

    resetState();
    emit("dismiss");
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(createdIssue.name),
        issueId: extractIssueUID(createdIssue.name),
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: String(error),
    });
  } finally {
    state.creating = false;
  }
};
</script>
