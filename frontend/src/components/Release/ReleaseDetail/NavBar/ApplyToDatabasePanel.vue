<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent :title="$t('changelist.apply-to-database')">
      <template #default>
        <div class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)]">
          <DatabaseAndGroupSelector
            :project="project"
            v-model:value="state.targetSelectState"
          />
        </div>
      </template>

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <div>
            <div
              v-if="
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
            <NButton quaternary @click.prevent="emit('close')">
              {{ $t("common.cancel") }}
            </NButton>
            <ErrorTipsButton
              style="--n-padding: 0 10px"
              :errors="createButtonErrors"
              :button-props="{
                type: 'primary',
              }"
              @click="handleCreate"
            >
              {{ $t("common.create") }}
            </ErrorTipsButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { planServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { getProjectNameReleaseId } from "@/store/modules/v1/common";
import { DatabaseGroupSchema } from "@/types/proto-es/v1/database_group_service_pb";
import { CreatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanSchema,
  Plan_ChangeDatabaseConfigSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { generateIssueTitle, issueV1Slug } from "@/utils";
import { useReleaseDetailContext } from "../context";
import { createIssueFromPlan } from "./utils";

const emit = defineEmits<{
  (event: "close"): void;
}>();

type LocalState = {
  targetSelectState?: DatabaseSelectState;
};

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const { release, project } = useReleaseDetailContext();

const state = reactive<LocalState>({});

const createButtonErrors = computed(() => {
  const errors: string[] = [];
  if (
    !state.targetSelectState ||
    (state.targetSelectState.changeSource === "DATABASE" &&
      state.targetSelectState.selectedDatabaseNameList.length === 0) ||
    (state.targetSelectState.changeSource === "GROUP" &&
      !state.targetSelectState.selectedDatabaseGroup)
  ) {
    errors.push("Please select at least one database");
  }
  return errors;
});

const handleCreate = async () => {
  if (!state.targetSelectState) {
    return;
  }

  const databaseList = state.targetSelectState.selectedDatabaseNameList.map(
    (name) => databaseStore.getDatabaseByName(name)
  );
  const databaseGroup = create(
    DatabaseGroupSchema,
    dbGroupStore.getDBGroupByName(
      state.targetSelectState.selectedDatabaseGroup || ""
    )
  );
  const newPlan = create(PlanSchema, {
    title: `Release "${release.value.title}"`,
    description: `Apply release "${release.value.title}" to selected databases.`,
    specs: [
      {
        id: crypto.randomUUID(),
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            targets:
              (state.targetSelectState.changeSource === "DATABASE"
                ? state.targetSelectState.selectedDatabaseNameList
                : [state.targetSelectState.selectedDatabaseGroup!]) || [],
            release: release.value.name,
          }),
        },
      },
    ],
  });
  const request = create(CreatePlanRequestSchema, {
    parent: project.value.name,
    plan: newPlan,
  });
  const response = await planServiceClientConnect.createPlan(request);
  const createdIssue = await createIssueFromPlan(project.value.name, {
    ...response,
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
      projectId: getProjectNameReleaseId(release.value.name)[0],
      issueSlug: issueV1Slug(createdIssue.name, createdIssue.title),
    },
  });
};
</script>
