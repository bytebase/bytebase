<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent :title="$t('common.apply-to-database')">
      <template #default>
        <div class="w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)]">
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
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import {
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { getProjectNameReleaseId } from "@/store/modules/v1/common";
import {
  CreatePlanRequestSchema,
  Plan_ChangeDatabaseConfigSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { useReleaseDetailContext } from "../context";

const emit = defineEmits<{
  (event: "close"): void;
}>();

type LocalState = {
  targetSelectState?: DatabaseSelectState;
};

const router = useRouter();
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

  // Determine enableGhost from release files
  const firstFile = release.value.files?.[0];
  const enableGhost = firstFile?.enableGhost ?? false;

  const newPlan = create(PlanSchema, {
    title: `Release "${release.value.title}"`,
    description: `Apply release "${release.value.title}" to selected databases.`,
    specs: [
      {
        id: uuidv4(),
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            targets:
              (state.targetSelectState.changeSource === "DATABASE"
                ? state.targetSelectState.selectedDatabaseNameList
                : [state.targetSelectState.selectedDatabaseGroup!]) || [],
            release: release.value.name,

            enableGhost,
          }),
        },
      },
    ],
  });

  // Create plan
  const planRequest = create(CreatePlanRequestSchema, {
    parent: project.value.name,
    plan: newPlan,
  });
  const createdPlan = await planServiceClientConnect.createPlan(planRequest);

  // Create rollout directly without creating an issue
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: createdPlan.name,
  });
  await rolloutServiceClientConnect.createRollout(rolloutRequest);

  // Extract planId from plan name (format: projects/{project}/plans/{planId})
  const planId = createdPlan.name.split("/").pop();

  // Redirect to rollout view
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
    params: {
      projectId: getProjectNameReleaseId(release.value.name)[0],
      planId,
    },
  });
};
</script>
