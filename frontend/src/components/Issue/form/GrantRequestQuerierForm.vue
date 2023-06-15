<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 gap-4 mb-4"
  >
    <div v-if="create" class="w-full flex flex-col justify-start items-start">
      <span class="flex w-full items-center textlabel mb-2">
        {{ $t("common.project") }}
        <RequiredStar />
      </span>
      <ProjectSelect
        class="!w-60"
        :only-userself="false"
        :selected-id="projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex w-full items-center textlabel mb-2">
        {{ $t("common.databases") }}
        <RequiredStar />
      </span>
      <div v-if="create" class="w-full">
        <NRadioGroup
          v-model:value="state.allDatabases"
          class="w-full !flex flex-row justify-start items-start gap-4"
          name="radiogroup"
        >
          <NTooltip trigger="hover">
            <template #trigger>
              <NRadio
                class="!leading-6 whitespace-nowrap"
                :value="true"
                :label="$t('issue.grant-request.all-databases')"
              />
            </template>
            {{ $t("issue.grant-request.all-databases-tip") }}
          </NTooltip>
          <div class="flex flex-row justify-start flex-wrap gap-y-2">
            <NRadio
              class="!leading-6"
              :value="false"
              :disabled="!state.projectId"
              :label="$t('issue.grant-request.manually-select')"
            />
            <button
              v-if="state.projectId && state.allDatabases === false"
              class="ml-2 normal-link h-6 disabled:cursor-not-allowed"
              @click="state.showSelectDatabasePanel = true"
            >
              {{ $t("common.select") }}
            </button>
          </div>
        </NRadioGroup>
        <div
          v-if="
            state.selectedDatabaseResourceList.length > 0 &&
            state.allDatabases === false
          "
          class="w-full max-w-3xl mt-2"
        >
          <DatabaseResourceTable
            class="w-full"
            :database-resource-list="state.selectedDatabaseResourceList"
          />
        </div>
      </div>
      <div
        v-else
        class="w-full flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
      >
        <span v-if="state.selectedDatabaseResourceList.length === 0">{{
          $t("issue.grant-request.all-databases")
        }}</span>
        <DatabaseResourceTable
          v-else
          class="w-full"
          :database-resource-list="state.selectedDatabaseResourceList"
        />
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex w-full items-center textlabel mb-4">
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
          class="!grid grid-cols-4 gap-4"
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

  <BBModal
    v-if="state.showSelectDatabasePanel && state.projectId"
    class="relative overflow-hidden"
    :title="$t('issue.grant-request.manually-select')"
    @close="state.showSelectDatabasePanel = false"
  >
    <SelectDatabaseResourceForm
      :project-id="state.projectId"
      :selected-database-resource-list="state.selectedDatabaseResourceList"
      @close="state.showSelectDatabasePanel = false"
      @update="handleSelectedDatabaseResourceChanged"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head } from "lodash-es";
import { NRadioGroup, NRadio, NInputNumber, NTooltip } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueLogic } from "../logic";
import {
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  PresetRoleType,
  UNKNOWN_ID,
} from "@/types";
import { extractUserUID, memberListInProjectV1 } from "@/utils";
import { useDatabaseV1Store, useProjectV1Store } from "@/store";
import { convertFromCELString } from "@/utils/issue/cel";
import { DatabaseResource } from "./SelectDatabaseResourceForm/common";
import RequiredStar from "@/components/RequiredStar.vue";
import SelectDatabaseResourceForm from "./SelectDatabaseResourceForm/index.vue";
import DatabaseResourceTable from "../table/DatabaseResourceTable.vue";

interface LocalState {
  showSelectDatabasePanel: boolean;
  // For creating
  projectId?: string;
  allDatabases: boolean;
  selectedDatabaseResourceList: DatabaseResource[];
  expireDays: number;
  customDays: number;
  // For reviewing
  expiredAt: string;
}

const { t } = useI18n();
const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  showSelectDatabasePanel: false,
  projectId: undefined,
  allDatabases: true,
  selectedDatabaseResourceList: [],
  expireDays: 7,
  customDays: 7,
  expiredAt: "",
});

const projectId = computed(() => {
  return create.value ? state.projectId : (issue.value as Issue).project.id;
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
    const userUID = extractUserUID(projectOwner.user.name);
    (issue.value as IssueCreate).assigneeId = Number(userUID);
  }
  state.selectedDatabaseResourceList =
    state.selectedDatabaseResourceList.filter((resource) => {
      const database = databaseStore.getDatabaseByName(resource.databaseName);
      return database.projectEntity.uid === projectId;
    });
};

const handleSelectedDatabaseResourceChanged = (
  databaseResourceList: DatabaseResource[]
) => {
  state.selectedDatabaseResourceList = databaseResourceList;
  state.showSelectDatabasePanel = false;
  state.allDatabases = false;

  if (create.value) {
    (
      (issue.value as IssueCreate).createContext as GrantRequestContext
    ).databaseResources = databaseResourceList;
  }
};

watch(
  () => [state.expireDays, state.customDays],
  () => {
    if (create.value) {
      const context = (issue.value as IssueCreate)
        .createContext as GrantRequestContext;
      if (state.expireDays === -1) {
        context.expireDays = state.customDays;
      } else {
        context.expireDays = state.expireDays;
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

      const conditionExpression = await convertFromCELString(
        payload.condition.expression
      );
      if (conditionExpression.expiredTime !== undefined) {
        state.expiredAt = dayjs(
          new Date(conditionExpression.expiredTime)
        ).format("LLL");
      }
      if (
        conditionExpression.databaseResources !== undefined &&
        conditionExpression.databaseResources.length > 0
      ) {
        state.selectedDatabaseResourceList =
          conditionExpression.databaseResources;
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
