<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent :title="$t('changelist.apply-to-database')">
      <template #default>
        <div class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)]">
          <StepTab
            :step-list="stepList"
            :current-index="state.currentStep"
            :show-footer="false"
          >
            <template #0>
              <DatabaseAndGroupSelector
                :project="project"
                :database-select-state="state.targetSelectState"
                @update="handleTargetChange"
              />
            </template>
            <template #1>
              <div
                v-if="!isRequesting && state.previewPlanResult"
                class="space-y-4"
              >
                <PreviewPlanDetail
                  :release="release"
                  :preview-plan-result="state.previewPlanResult"
                  :database-select-state="state.targetSelectState"
                  :allow-out-of-order="state.allowOutOfOrder"
                />
              </div>
              <div
                v-else
                class="flex items-center justify-center py-4 text-gray-400 text-sm"
              >
                <BBSpin />
              </div>
            </template>
          </StepTab>
        </div>
      </template>

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <div>
            <div
              v-if="
                state.currentStep === 0 &&
                state.targetSelectState?.changeSource === 'DATABASE' &&
                state.targetSelectState?.selectedDatabaseNameList.length > 0
              "
              class="textinfolabel"
            >
              {{
                $t("database.selected-n-databases", {
                  n: state.targetSelectState?.selectedDatabaseNameList.length,
                })
              }}
            </div>
          </div>

          <div class="flex items-center justify-end gap-x-3">
            <NCheckbox
              v-if="state.currentStep === 1"
              v-model:checked="state.allowOutOfOrder"
            >
              {{ $t("release.allow-out-of-order") }}
            </NCheckbox>
            <NButton quaternary @click.prevent="handleCancelClick">
              {{
                state.currentStep === 0
                  ? $t("common.cancel")
                  : $t("common.back")
              }}
            </NButton>
            <ErrorTipsButton
              style="--n-padding: 0 10px"
              :errors="nextButtonErrors"
              :button-props="{
                type: 'primary',
              }"
              @click="handleClickNext"
            >
              {{ $t("common.next") }}
            </ErrorTipsButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NCheckbox } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { StepTab } from "@/components/v2";
import { planServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import {
  PreviewPlanResponse,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import {
  extractProjectResourceName,
  generateIssueTitle,
  issueV1Slug,
} from "@/utils";
import { useReleaseDetailContext } from "../context";
import PreviewPlanDetail from "./PreviewPlanDetail.vue";
import { createIssueFromPlan } from "./utils";

const emit = defineEmits<{
  (event: "close"): void;
}>();

type LocalState = {
  currentStep: number;
  allowOutOfOrder: boolean;
  targetSelectState?: DatabaseSelectState;
  previewPlanResult?: PreviewPlanResponse;
};

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const { release, project } = useReleaseDetailContext();
const isRequesting = ref(false);

const state = reactive<LocalState>({
  currentStep: 0,
  allowOutOfOrder: false,
});

const stepList = computed(() => [
  { title: t("database.sync-schema.select-target-databases") },
  { title: t("common.preview") },
]);

const flattenSpecList = computed((): Plan_Spec[] => {
  return (
    state.previewPlanResult?.plan?.steps.flatMap((step) => {
      return step.specs;
    }) || []
  );
});

const nextButtonErrors = computed(() => {
  const errors: string[] = [];
  if (state.currentStep === 0) {
    if (
      !state.targetSelectState ||
      (state.targetSelectState.changeSource === "DATABASE" &&
        state.targetSelectState.selectedDatabaseNameList.length === 0) ||
      (state.targetSelectState.changeSource === "GROUP" &&
        !state.targetSelectState.selectedDatabaseGroup)
    ) {
      errors.push("Please select at least one database");
    }
  } else if (state.currentStep === 1) {
    if (!state.previewPlanResult) {
      errors.push("Failed to preview plan");
    }
    if (flattenSpecList.value.length === 0) {
      errors.push("No plan to apply");
    }
  }
  return errors;
});

const handleTargetChange = (databaseSelectState: DatabaseSelectState) => {
  state.targetSelectState = databaseSelectState;
};

const handleCancelClick = () => {
  if (state.currentStep === 0) {
    emit("close");
  } else {
    state.currentStep = 0;
    state.previewPlanResult = undefined;
  }
};

const handleClickNext = async () => {
  if (state.currentStep === 0) {
    await previewPlan();
  } else if (state.currentStep === 1) {
    if (!state.targetSelectState || !state.previewPlanResult) {
      return;
    }
    const databaseList = state.targetSelectState.selectedDatabaseNameList.map(
      (name) => databaseStore.getDatabaseByName(name)
    );
    const databaseGroup = DatabaseGroup.fromPartial({
      ...dbGroupStore.getDBGroupByName(
        state.targetSelectState.selectedDatabaseGroup || ""
      ),
    });
    const createdPlan = await planServiceClient.createPlan({
      parent: project.value.name,
      plan: state.previewPlanResult.plan,
    });
    const createdIssue = await createIssueFromPlan(project.value.name, {
      ...createdPlan,
      // Override title and description.
      title: generateIssueTitle(
        "bb.issue.database.schema.update",
        state.targetSelectState.changeSource === "DATABASE"
          ? databaseList.map((db) => db.databaseName)
          : [databaseGroup?.title]
      ),
      description: `Apply release "${release.value.title}"`,
    });
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(release.value.project),
        issueSlug: issueV1Slug(createdIssue),
      },
    });
  }
};

const previewPlan = async () => {
  if (!state.targetSelectState) {
    return;
  }

  isRequesting.value = true;
  const resp = await planServiceClient.previewPlan({
    project: project.value.name,
    release: release.value.name,
    targets:
      state.targetSelectState.changeSource === "DATABASE"
        ? state.targetSelectState.selectedDatabaseNameList
        : [state.targetSelectState.selectedDatabaseGroup!],
    allowOutOfOrder: state.allowOutOfOrder,
  });
  state.previewPlanResult = resp;
  state.currentStep = 1;
  isRequesting.value = false;
};

watch(
  () => state.allowOutOfOrder,
  async () => {
    if (state.currentStep === 1) {
      await previewPlan();
    }
  }
);
</script>
