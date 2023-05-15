<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8 gap-4"
  >
    <div v-if="create" class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center">
        {{ $t("common.project") }}
        <RequiredStar />
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :only-userself="false"
        :selected-id="projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center">
        {{ $t("common.databases") }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.allDatabases"
          class="w-full !flex flex-row justify-start items-center gap-4"
          name="radiogroup"
        >
          <NRadio
            :value="true"
            :label="$t('issue.grant-request.all-databases')"
          />
          <div>
            <NRadio
              :value="false"
              :disabled="!state.projectId"
              :label="$t('issue.grant-request.manually-select')"
            />
            <button
              v-if="state.projectId"
              class="ml-2 normal-link disabled:cursor-not-allowed"
              @click="state.showSelectDatabasePanel = true"
            >
              {{ $t("common.select") }}
            </button>
          </div>
        </NRadioGroup>
      </div>
      <div v-else class="flex flex-row justify-start items-start gap-4">
        <span v-if="state.selectedDatabaseIdList.length === 0">{{
          $t("issue.grant-request.all-databases")
        }}</span>
        <div
          v-for="database in selectedDatabaseList"
          v-else
          :key="database.id"
          class="flex flex-row justify-start items-center"
        >
          <InstanceEngineIcon class="mr-1" :instance="database.instance" />
          {{ database.name }}
        </div>
      </div>
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-start">
        {{
          create
            ? $t("issue.grant-request.expire-days")
            : $t("issue.grant-request.expired-at")
        }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.expireDays"
          class="!grid grid-cols-4 gap-2"
          name="radiogroup"
        >
          <NRadio
            v-for="day in expireDaysOptions"
            :key="day.value"
            :value="day.value"
            :label="day.label"
          />
          <div class="col-span-2 flex flex-row justify-start items-center">
            <NRadio :value="-1" :label="$t('issue.grant-request.customize')" />
            <NInputNumber
              v-model:value="state.customDays"
              :disabled="state.expireDays !== -1"
              size="small"
              :min="1"
              :max="365"
              :step="1"
              class="!w-24"
            />
          </div>
        </NRadioGroup>
      </div>
      <div v-else>
        {{ state.expiredAt }}
      </div>
    </div>
  </div>

  <DatabasesSelectPanel
    v-if="state.showSelectDatabasePanel && state.projectId"
    :project-id="state.projectId"
    :selected-database-id-list="state.selectedDatabaseIdList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseIdListChanged"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NRadioGroup, NRadio, NInputNumber } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useIssueLogic } from "../logic";
import {
  DatabaseId,
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  ProjectId,
  UNKNOWN_ID,
} from "@/types";
import { getProjectMemberList, parseExpiredTimeString } from "@/utils";
import DatabasesSelectPanel from "../../DatabasesSelectPanel.vue";
import { useDatabaseStore } from "@/store";
import { useInstanceV1Store } from "@/store/modules/v1/instance";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import RequiredStar from "@/components/RequiredStar.vue";

interface LocalState {
  showSelectDatabasePanel: boolean;
  // For creating
  projectId?: ProjectId;
  allDatabases: boolean;
  selectedDatabaseIdList: DatabaseId[];
  expireDays: number;
  customDays: number;
  // For reviewing
  expiredAt: string;
}

const expireDaysOptions = [
  {
    value: 7,
    label: "7 days",
  },
  {
    value: 30,
    label: "30 days",
  },
  {
    value: 60,
    label: "60 days",
  },
  {
    value: 90,
    label: "90 days",
  },
  {
    value: 180,
    label: "6 months",
  },
  {
    value: 365,
    label: "1 year",
  },
];

const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseStore();
const instanceV1Store = useInstanceV1Store();
const state = reactive<LocalState>({
  showSelectDatabasePanel: false,
  projectId: undefined,
  allDatabases: true,
  selectedDatabaseIdList: [],
  expireDays: 7,
  customDays: 7,
  expiredAt: "",
});

const projectId = computed(() => {
  return create.value ? state.projectId : (issue.value as Issue).project.id;
});

const selectedDatabaseList = computed(() => {
  return state.selectedDatabaseIdList.map((id) => {
    return databaseStore.getDatabaseById(id);
  });
});

onMounted(() => {
  if (create.value) {
    const projectId = (issue.value as IssueCreate).projectId;
    if (projectId && projectId !== UNKNOWN_ID) {
      handleProjectSelect(projectId);
    }
  }
});

const handleProjectSelect = async (projectId: ProjectId) => {
  if (!create.value) {
    return;
  }

  state.projectId = projectId;
  (issue.value as IssueCreate).projectId = projectId;
  // update issue assignee
  const projectOwner = head(
    (await getProjectMemberList(projectId)).filter(
      (member) => !member.roleList.includes("OWNER")
    )
  );
  if (projectOwner) {
    (issue.value as IssueCreate).assigneeId = projectOwner.principal.id;
  }
};

const handleSelectedDatabaseIdListChanged = (databaseIdList: DatabaseId[]) => {
  state.selectedDatabaseIdList = databaseIdList;
  state.showSelectDatabasePanel = false;
  state.allDatabases = false;
  if (create.value) {
    (
      (issue.value as IssueCreate).createContext as GrantRequestContext
    ).databases = databaseIdList as string[];
  }
};

watch(
  () => [state.expireDays, state.customDays],
  () => {
    if (create.value) {
      if (state.expireDays === -1) {
        (
          (issue.value as IssueCreate).createContext as GrantRequestContext
        ).expireDays = state.customDays;
      } else {
        (
          (issue.value as IssueCreate).createContext as GrantRequestContext
        ).expireDays = state.expireDays;
      }
    }
  }
);

watch(
  create,
  async () => {
    if (!create.value) {
      const payload = ((issue.value as Issue).payload as any)
        .grantRequest as GrantRequestPayload;
      if (payload.role !== "roles/QUERIER") {
        throw "Only support QUERIER role";
      }
      const expressionList = payload.condition.expression.split(" && ");
      for (const expression of expressionList) {
        const fields = expression.split(" ");
        if (fields[0] === "request.time") {
          state.expiredAt = parseExpiredTimeString(fields[2]).toLocaleString();
        } else if (fields[0] === "resource.database") {
          const databaseIdList = [];
          for (const url of JSON.parse(fields[2])) {
            const value = url.split("/");
            const instanceName = value[1] || "";
            const databaseName = value[3] || "";
            const instance = await instanceV1Store.getOrFetchInstanceByName(
              instanceNamePrefix + instanceName
            );
            const databaseList =
              await databaseStore.getOrFetchDatabaseListByInstanceId(
                instance.uid
              );
            const database = databaseList.find(
              (db) => db.name === databaseName
            );
            if (database) {
              databaseIdList.push(database.id);
            }
          }
          state.selectedDatabaseIdList = databaseIdList;
        }
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
