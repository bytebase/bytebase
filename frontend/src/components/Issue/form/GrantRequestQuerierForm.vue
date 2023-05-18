<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8 gap-4"
  >
    <div v-if="create" class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center textlabel !leading-6 mt-2">
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
      <span class="flex w-40 items-center textlabel !leading-6">
        {{ $t("common.databases") }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.allDatabases"
          class="w-full !flex flex-row justify-start items-start gap-4"
          name="radiogroup"
        >
          <NRadio
            class="!leading-6 whitespace-nowrap"
            :value="true"
            :label="$t('issue.grant-request.all-databases')"
          />
          <div class="flex flex-row justify-start flex-wrap">
            <NRadio
              class="!leading-6"
              :value="false"
              :disabled="!state.projectId"
              :label="$t('issue.grant-request.manually-select')"
              @click="handleManuallySelectClick"
            />
            <button
              v-if="state.projectId"
              class="ml-2 normal-link h-6 disabled:cursor-not-allowed"
              @click="state.showSelectDatabasePanel = true"
            >
              {{ $t("common.select") }}
            </button>
            <div
              v-if="selectedDatabaseList.length > 0"
              class="ml-6 flex flex-row justify-start items-start gap-4"
            >
              <div
                v-for="database in selectedDatabaseList"
                :key="database.id"
                class="flex flex-row justify-start items-center"
              >
                <InstanceEngineIcon
                  class="mr-1"
                  :instance="database.instance"
                />
                {{ database.name }}
              </div>
            </div>
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
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-start textlabel !leading-6">
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
          class="!grid grid-cols-4 gap-x-4 gap-y-4"
          name="radiogroup"
        >
          <div
            v-for="day in expireDaysOptions"
            :key="day.value"
            class="col-span-1 flex flex-row justify-start items-center"
          >
            <NRadio :value="day.value" :label="day.label" />
          </div>
          <div class="col-span-2 flex flex-row justify-start items-center">
            <NRadio :value="-1" :label="$t('issue.grant-request.customize')" />
            <NInputNumber
              v-model:value="state.customDays"
              class="!w-24 ml-2"
              :disabled="state.expireDays !== -1"
              :min="1"
              :show-button="false"
              :placeholder="''"
            >
              <template #suffix>{{ $t("common.date.days") }}</template>
            </NInputNumber>
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
import { head, uniq } from "lodash-es";
import { NRadioGroup, NRadio, NInputNumber } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueLogic } from "../logic";
import {
  DatabaseId,
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  PresetRoleType,
  UNKNOWN_ID,
} from "@/types";
import {
  getDatabaseIdByName,
  memberListInProjectV1,
  parseConditionExpressionString,
} from "@/utils";
import {
  convertUserToPrincipal,
  useDatabaseStore,
  useProjectV1Store,
} from "@/store";
import RequiredStar from "@/components/RequiredStar.vue";
import DatabasesSelectPanel from "../../DatabasesSelectPanel.vue";

interface LocalState {
  showSelectDatabasePanel: boolean;
  // For creating
  projectId?: string;
  allDatabases: boolean;
  selectedDatabaseIdList: DatabaseId[];
  expireDays: number;
  customDays: number;
  // For reviewing
  expiredAt: string;
}

const { t } = useI18n();
const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseStore();
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

const expireDaysOptions = computed(() => [
  {
    value: 7,
    label: t("common.date.days", { days: 7 }),
  },
  {
    value: 30,
    label: t("common.date.days", { days: 30 }),
  },
  {
    value: 60,
    label: t("common.date.days", { days: 60 }),
  },
  {
    value: 90,
    label: t("common.date.days", { days: 90 }),
  },
  {
    value: 180,
    label: t("common.date.months", { months: 6 }),
  },
  {
    value: 365,
    label: t("common.date.years", { years: 1 }),
  },
]);

onMounted(() => {
  if (create.value) {
    const projectId = String((issue.value as IssueCreate).projectId);
    if (projectId && projectId !== String(UNKNOWN_ID)) {
      handleProjectSelect(projectId);
    }
  }
});

const handleProjectSelect = async (projectId: string) => {
  if (!create.value) {
    return;
  }

  state.projectId = projectId;
  (issue.value as IssueCreate).projectId = parseInt(projectId, 10);
  // update issue assignee

  const project = await useProjectV1Store().getOrFetchProjectByUID(projectId);
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const ownerList = memberList.filter((member) =>
    member.roleList.includes(PresetRoleType.OWNER)
  );
  const projectOwner = head(ownerList);
  if (projectOwner) {
    const ownerPrincipal = convertUserToPrincipal(projectOwner.user);
    (issue.value as IssueCreate).assigneeId = ownerPrincipal.id;
  }
  state.selectedDatabaseIdList = state.selectedDatabaseIdList.filter((id) => {
    const database = databaseStore.getDatabaseById(id);
    return String(database.project.id) === projectId;
  });
};

const handleManuallySelectClick = () => {
  if (state.selectedDatabaseIdList.length === 0) {
    state.showSelectDatabasePanel = true;
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
      if (payload.role !== PresetRoleType.QUERIER) {
        throw "Only support QUERIER role";
      }

      const conditionExpression = parseConditionExpressionString(
        payload.condition.expression
      );
      if (conditionExpression.expiredTime !== undefined) {
        state.expiredAt = new Date(
          conditionExpression.expiredTime
        ).toLocaleString();
      }
      if (
        conditionExpression.databases !== undefined &&
        conditionExpression.databases.length > 0
      ) {
        const databaseIdList = [];
        for (const databaseName of conditionExpression.databases) {
          const id = await getDatabaseIdByName(databaseName);
          if (id && id !== UNKNOWN_ID) {
            databaseIdList.push(id);
          }
        }
        state.selectedDatabaseIdList = uniq(databaseIdList);
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
