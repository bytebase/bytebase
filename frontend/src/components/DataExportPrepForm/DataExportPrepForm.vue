<template>
  <DrawerContent class="max-w-[100vw]">
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>
          {{ $t("custom-approval.risk-rule.risk.namespace.data_export") }}
        </span>
      </div>
    </template>

    <div
      class="flex flex-col gap-y-4 h-full w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <DatabaseAndGroupSelector
        :project="project"
        v-model:value="state.targetSelectState"
      />
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div></div>

        <div class="flex items-center justify-end gap-x-3">
          <NButton @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!validSelectState"
            @click="navigateToIssuePage"
          >
            {{ $t("common.next") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { type LocationQueryRaw, useRouter } from "vue-router";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useProjectByName } from "@/store";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";
import { DrawerContent } from "../v2";

type LocalState = {
  targetSelectState?: DatabaseSelectState;
};

const props = defineProps<{
  projectName: string;
}>();

const { project } = useProjectByName(computed(() => props.projectName));

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const state = reactive<LocalState>({});

const validSelectState = computed(() => {
  if (!state.targetSelectState) {
    return false;
  }
  if (state.targetSelectState.changeSource === "DATABASE") {
    return state.targetSelectState.selectedDatabaseNameList.length > 0;
  }
  return !!state.targetSelectState.selectedDatabaseGroup;
});

const navigateToIssuePage = async () => {
  if (!state.targetSelectState) {
    return;
  }

  const issueType = "bb.issue.database.data.export";
  const query: LocationQueryRaw = {
    template: issueType,
    name: generateIssueTitle(issueType),
  };

  if (state.targetSelectState.changeSource === "DATABASE") {
    query["databaseList"] =
      state.targetSelectState.selectedDatabaseNameList.join(",");
  } else {
    query["databaseGroupName"] = state.targetSelectState.selectedDatabaseGroup;
  }

  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      planId: "create",
    },
    query,
  });
};

const cancel = () => {
  emit("dismiss");
};
</script>
