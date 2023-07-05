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
          <div class="w-full mb-2">
            <NRadioGroup
              v-model:value="state.exportMethod"
              class="w-full !flex flex-row justify-start items-center gap-4"
              name="export-method"
            >
              <NRadio :value="'SQL'" label="SQL" />
              <NTooltip :disabled="allowSelectTableResource">
                <template #trigger>
                  <NRadio
                    :disabled="!allowSelectTableResource"
                    :value="'DATABASE'"
                    :label="$t('common.database')"
                  />
                </template>
                {{ $t("issue.grant-request.please-select-database-first") }}
              </NTooltip>
            </NRadioGroup>
          </div>
          <div
            v-show="state.exportMethod === 'SQL'"
            class="w-full h-[300px] border rounded"
          >
            <MonacoEditor
              class="w-full h-full py-2"
              :value="state.statement"
              :auto-focus="false"
              :language="'sql'"
              :dialect="dialect"
              @change="handleStatementChange"
            />
          </div>
          <div
            v-if="state.exportMethod === 'DATABASE'"
            class="w-full flex flex-row justify-start items-center"
          >
            <SelectTableForm
              :project-id="(state.projectId as string)"
              :database-id="state.databaseId as string"
              :selected-database-resource-list="
                selectedTableResource ? [selectedTableResource] : []
              "
              @update="handleTableResourceUpdate"
            />
          </div>
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
          <span class="flex items-center textlabel mb-2">
            {{ $t("issue.grant-request.export-format") }}
            <RequiredStar />
          </span>
          <div>
            <NRadioGroup
              v-model:value="state.exportFormat"
              class="w-full !flex flex-row justify-start items-center gap-4"
              name="export-format"
            >
              <NRadio :value="'CSV'" label="CSV" />
              <NRadio :value="'JSON'" label="JSON" />
              <NRadio :value="'SQL'" label="SQL" />
            </NRadioGroup>
          </div>
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-start textlabel mb-2">
            {{ $t("issue.grant-request.expire-days") }}
            <RequiredStar />
          </span>
          <div>
            <NRadioGroup
              v-model:value="state.expireDays"
              class="!grid grid-cols-6 gap-4"
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
                <NRadio
                  :value="-1"
                  :label="$t('issue.grant-request.customize')"
                />
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
import {
  NDrawer,
  NDrawerContent,
  NRadioGroup,
  NRadio,
  NInputNumber,
  NInput,
  NTooltip,
} from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  DatabaseResource,
  IssueCreate,
  PresetRoleType,
  SQLDialect,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import {
  extractUserUID,
  instanceV1Name,
  issueSlug,
  memberListInProjectV1,
} from "@/utils";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useIssueStore,
  useProjectV1Store,
} from "@/store";
import MonacoEditor from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import DatabaseSelect from "@/components/DatabaseSelect.vue";
import { Engine } from "@/types/proto/v1/common";
import { head } from "lodash-es";
import { useRouter } from "vue-router";
import SelectTableForm from "./SelectTableForm/index.vue";
import dayjs from "dayjs";
import { stringifyDatabaseResources } from "@/utils/issue/cel";

interface LocalState {
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  expireDays: number;
  customDays: number;
  maxRowCount: number;
  exportMethod: "SQL" | "DATABASE";
  exportFormat: "CSV" | "JSON" | "SQL";
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
  expireDays: 1,
  customDays: 7,
  maxRowCount: 1000,
  exportMethod: "SQL",
  exportFormat: "CSV",
  statement: "",
  description: "",
});
const selectedTableResource = ref<DatabaseResource>();

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

  if (state.exportMethod === "SQL") {
    return state.statement && state.statement !== "";
  } else {
    return selectedTableResource.value !== undefined;
  }
});

const allowSelectTableResource = computed(() => {
  return state.databaseId !== undefined;
});

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine ?? Engine.MYSQL);
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

const handleTableResourceUpdate = (
  databaseResourceList: DatabaseResource[]
) => {
  if (databaseResourceList.length > 1) {
    throw new Error("Only one table can be selected");
  } else if (databaseResourceList.length === 0) {
    selectedTableResource.value = undefined;
  } else {
    selectedTableResource.value = databaseResourceList[0];
  }
};

const handleStatementChange = (value: string) => {
  state.statement = value;
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
  const expireDays =
    state.expireDays === -1 ? state.customDays : state.expireDays;
  expression.push(
    `request.time < timestamp("${dayjs()
      .add(expireDays, "days")
      .toISOString()}")`
  );
  expression.push(`request.export_format == "${state.exportFormat}"`);
  expression.push(`request.row_limit == ${state.maxRowCount}`);
  if (state.exportMethod === "SQL") {
    expression.push(`request.statement == "${btoa(state.statement)}"`);
    const cel = stringifyDatabaseResources([
      {
        databaseName: selectedDatabase.value!.name,
      },
    ]);
    expression.push(cel);
  } else {
    if (!selectedTableResource.value) {
      throw new Error("No table selected");
    }
    const cel = stringifyDatabaseResources([selectedTableResource.value]);
    expression.push(cel);
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

  if (state.exportMethod === "SQL") {
    return `Request data export for "${database.databaseName} (${database.instanceEntity.title})"`;
  } else {
    const tableResource = selectedTableResource.value as DatabaseResource;
    const nameList = [database.databaseName];
    if (tableResource.schema) {
      nameList.push(tableResource.schema);
    }
    if (tableResource.table) {
      nameList.push(tableResource.table);
    }
    return `Request data export for "${nameList.join(".")} (${
      database.instanceEntity.title
    })"`;
  }
};
</script>
