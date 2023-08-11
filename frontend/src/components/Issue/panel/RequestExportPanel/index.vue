<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('quick-action.request-export')"
      :closable="true"
      class="w-[50rem] max-w-[100vw] relative"
    >
      <div class="w-full mx-auto space-y-4">
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("common.project") }}
            <RequiredStar />
          </span>
          <ProjectSelect
            class="!w-60 shrink-0"
            :only-userself="false"
            :selected-id="state.projectId"
            @select-project-id="handleProjectSelect"
          />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("common.database") }}
            <RequiredStar />
          </span>
          <div class="flex flex-row justify-start items-center">
            <EnvironmentSelect
              class="!w-60 mr-4 shrink-0"
              name="environment"
              :select-default="false"
              :selected-id="state.environmentId"
              @select-environment-id="handleEnvironmentSelect"
            />
            <DatabaseSelect
              class="!w-96"
              :selected-id="state.databaseId ?? String(UNKNOWN_ID)"
              :mode="'ALL'"
              :environment-id="state.environmentId"
              :project-id="state.projectId"
              :sync-status="'OK'"
              :customize-item="true"
              @select-database-id="handleDatabaseSelect"
            >
              <template #customizeItem="{ database }">
                <div class="flex items-center">
                  <InstanceV1EngineIcon :instance="database.instanceEntity" />
                  <span class="mx-2">{{ database.databaseName }}</span>
                  <span class="text-gray-400">
                    ({{ instanceV1Name(database.instanceEntity) }})
                  </span>
                </div>
              </template>
            </DatabaseSelect>
          </div>
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("issue.grant-request.export-method") }}
            <RequiredStar />
          </span>
          <ExportResourceForm
            :project-id="state.projectId"
            :database-id="state.databaseId"
            :statement="statement"
            @update:condition="state.databaseResourceCondition = $event"
            @update:database-resources="state.databaseResources = $event"
          />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("issue.grant-request.export-rows") }}
            <RequiredStar />
          </span>
          <input
            v-model="state.maxRowCount"
            required
            type="number"
            class="textfield"
            placeholder="Max row count"
          />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-start textlabel mb-2">
            {{ $t("common.expiration") }}
            <RequiredStar />
          </span>
          <ExpirationSelector
            class="grid-cols-6"
            :options="expireDaysOptions"
            :value="state.expireDays"
            @update="state.expireDays = $event"
          />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">{{
            $t("common.reason")
          }}</span>
          <NInput
            v-model:value="state.description"
            type="textarea"
            class="w-full"
            placeholder=""
          />
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton
            type="primary"
            :disabled="!allowCreate"
            @click="doCreateIssue"
          >
            {{ $t("common.ok") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head, isUndefined } from "lodash-es";
import { NDrawer, NDrawerContent, NInput } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseSelect from "@/components/DatabaseSelect.vue";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useIssueStore,
  useProjectV1Store,
} from "@/store";
import {
  DatabaseResource,
  IssueCreate,
  PresetRoleType,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
} from "@/types";
import {
  extractUserUID,
  instanceV1Name,
  issueSlug,
  memberListInProjectV1,
} from "@/utils";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import ExportResourceForm from "./ExportResourceForm/index.vue";

interface LocalState {
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  databaseResourceCondition?: string;
  databaseResources: DatabaseResource[];
  expireDays: number;
  maxRowCount: number;
  statement: string;
  description: string;
}

const props = defineProps<{
  databaseId?: string;
  statement?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUserV1();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  databaseResources: [],
  expireDays: 1,
  maxRowCount: 1000,
  statement: "",
  description: "",
});

const selectedDatabase = computed(() => {
  if (!state.databaseId || state.databaseId === String(UNKNOWN_ID)) {
    return undefined;
  }
  return databaseStore.getDatabaseByUID(state.databaseId);
});

const expireDaysOptions = computed(() => [
  {
    value: 1,
    label: t("common.date.days", { days: 1 }),
  },
  {
    value: 3,
    label: t("common.date.days", { days: 3 }),
  },
  {
    value: 7,
    label: t("common.date.days", { days: 7 }),
  },
  {
    value: 15,
    label: t("common.date.days", { days: 15 }),
  },
]);

const allowCreate = computed(() => {
  if (!state.databaseId) {
    return false;
  }
  if (isUndefined(state.databaseResourceCondition)) {
    return false;
  }
  return true;
});

onMounted(async () => {
  if (props.databaseId) {
    handleDatabaseSelect(props.databaseId);
  }
  if (props.statement) {
    state.statement = props.statement;
  }
});

const handleProjectSelect = async (projectId: string) => {
  state.projectId = projectId;
};

const handleEnvironmentSelect = (environmentId: string) => {
  state.environmentId = environmentId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  // Unselect database if it doesn't belong to the newly selected environment.
  if (
    database &&
    database.uid !== String(UNKNOWN_ID) &&
    database.instanceEntity.environmentEntity.uid !== state.environmentId
  ) {
    state.databaseId = undefined;
  }
};

const handleDatabaseSelect = (databaseId: string) => {
  state.databaseId = databaseId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  if (database && database.uid !== String(UNKNOWN_ID)) {
    handleProjectSelect(database.projectEntity.uid);
    handleEnvironmentSelect(database.instanceEntity.environmentEntity.uid);
  }
};

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const newIssue: IssueCreate = {
    name: generateIssueName(),
    type: "bb.issue.grant.request",
    description: state.description,
    projectId: Number(state.projectId),
    assigneeId: SYSTEM_BOT_ID,
    createContext: {},
    payload: {},
  };

  // update issue's assignee to first project owner.
  const project = await useProjectV1Store().getOrFetchProjectByUID(
    state.projectId!
  );
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const ownerList = memberList.filter((member) =>
    member.roleList.includes(PresetRoleType.OWNER)
  );
  const projectOwner = head(ownerList);
  if (projectOwner) {
    const userUID = extractUserUID(projectOwner.user.name);
    newIssue.assigneeId = Number(userUID);
  }

  const expression: string[] = [];
  const expireDays = state.expireDays;
  expression.push(
    `request.time < timestamp("${dayjs()
      .add(expireDays, "days")
      .toISOString()}")`
  );
  expression.push(`request.row_limit <= ${state.maxRowCount}`);
  if (state.databaseResourceCondition) {
    expression.push(state.databaseResourceCondition);
  }
  // If the export statement is not empty, add the selected database to the condition.
  if (state.databaseResources.length === 0) {
    const condition = stringifyDatabaseResources([
      {
        databaseName: selectedDatabase.value!.name,
      },
    ]);
    expression.push(condition);
  }

  const celExpressionString = expression.join(" && ");
  newIssue.payload = {
    grantRequest: {
      role: "roles/EXPORTER",
      user: currentUser.value.name,
      condition: {
        expression: celExpressionString,
      },
      // We need to pass a string type value to the expiration field because
      // the type of Duration proto is string.
      expiration: `${expireDays * 24 * 60 * 60}s`,
    },
  };

  const issue = await useIssueStore().createIssue(newIssue);
  router.push(`/issue/${issueSlug(issue.name, issue.id)}`);
};

const generateIssueName = () => {
  const database = selectedDatabase.value;
  if (!database) {
    throw new Error("No database selected");
  }

  if (state.databaseResources.length === 0) {
    return `Request data export for "${database.databaseName} (${database.instanceEntity.title})"`;
  } else {
    const sections: string[] = [];
    for (const databaseResource of state.databaseResources) {
      const nameList = [database.databaseName];
      if (databaseResource.schema) {
        nameList.push(databaseResource.schema);
      }
      if (databaseResource.table) {
        nameList.push(databaseResource.table);
      }
      sections.push(nameList.join("."));
    }
    return `Request data export for "${sections.join(".")} (${
      database.instanceEntity.title
    })"`;
  }
};
</script>
