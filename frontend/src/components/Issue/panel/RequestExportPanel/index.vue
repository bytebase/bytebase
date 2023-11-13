<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('quick-action.request-export-permission')"
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
            :project="state.projectId"
            :filter-by-current-user="false"
            @update:project="handleProjectSelect"
          />
        </div>

        <template v-if="props.statementOnly">
          <div class="w-full flex flex-col justify-start items-start">
            <span class="flex items-center textlabel mb-2">
              {{ $t("common.database") }}
              <RequiredStar />
            </span>
            <div class="flex flex-row justify-start items-center">
              <EnvironmentSelect
                class="!w-60 mr-4 shrink-0"
                :environment="state.environmentId"
                @update:environment="handleEnvironmentSelect"
              />
              <DatabaseSelect
                class="!w-96"
                :database="state.databaseId"
                :environment="state.environmentId"
                :project="state.projectId"
                @update:database="handleDatabaseSelect"
              >
              </DatabaseSelect>
            </div>
          </div>
          <div class="w-full flex flex-col justify-start items-start">
            <span class="flex items-center textlabel mb-2">
              SQL
              <RequiredStar />
            </span>
            <div class="w-full h-[300px]">
              <MonacoEditorV2
                v-model:content="state.statement"
                class="w-full h-full rounded border"
                :auto-focus="false"
                :dialect="dialect"
              />
            </div>
          </div>
        </template>
        <template v-else>
          <div class="w-full flex flex-col justify-start items-start">
            <span class="flex items-center textlabel mb-2">
              {{ $t("common.databases") }}
              <RequiredStar />
            </span>
            <DatabaseResourceForm
              :project-id="state.projectId"
              :database-resources="state.databaseResources"
              @update:condition="state.databaseResourceCondition = $event"
              @update:database-resources="state.databaseResources = $event"
            />
          </div>
        </template>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("issue.grant-request.export-rows") }}
            <RequiredStar />
          </span>
          <NInputNumber
            v-model:value="state.maxRowCount"
            required
            class="!w-60"
            placeholder="Max row count"
            :min="1"
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
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NInput,
  NInputNumber,
} from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import { MonacoEditorV2 } from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  ProjectSelect,
  EnvironmentSelect,
  DatabaseSelect,
} from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  pushNotification,
} from "@/store";
import {
  DatabaseResource,
  PresetRoleType,
  SQLDialect,
  SYSTEM_BOT_EMAIL,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import { Engine } from "@/types/proto/v1/common";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import { issueSlug, memberListInProjectV1 } from "@/utils";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import DatabaseResourceForm from "../RequestQueryPanel/DatabaseResourceForm/index.vue";

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
  projectId?: string;
  databaseId?: string;
  statement?: string;
  statementOnly?: boolean;
  redirectToIssuePage?: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUserV1();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
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

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine ?? Engine.MYSQL);
});

const allowCreate = computed(() => {
  if (!state.projectId) {
    return false;
  }
  if (props.statementOnly) {
    if (!state.databaseId || !state.statement) {
      return false;
    }
  } else {
    if (isUndefined(state.databaseResourceCondition)) {
      return false;
    }
  }
  return true;
});

onMounted(async () => {
  if (props.projectId) {
    handleProjectSelect(props.projectId);
  }
  if (props.databaseId) {
    handleDatabaseSelect(props.databaseId);
  }
  if (props.statement) {
    state.statement = props.statement;
  }
});

const handleProjectSelect = async (projectId: string | undefined) => {
  state.projectId = projectId;
};

const handleEnvironmentSelect = (environmentId: string | undefined) => {
  state.environmentId = environmentId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  // Unselect database if it doesn't belong to the newly selected environment.
  if (
    database &&
    database.uid !== String(UNKNOWN_ID) &&
    database.effectiveEnvironmentEntity.uid !== state.environmentId
  ) {
    state.databaseId = undefined;
  }
};

const handleDatabaseSelect = (databaseId: string | undefined) => {
  state.databaseId = databaseId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  if (database && database.uid !== String(UNKNOWN_ID)) {
    handleProjectSelect(database.projectEntity.uid);
    handleEnvironmentSelect(database.effectiveEnvironmentEntity.uid);
  }
};

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const newIssue = Issue.fromPartial({
    title: generateIssueName(),
    description: state.description,
    type: Issue_Type.GRANT_REQUEST,
    assignee: `users/${SYSTEM_BOT_EMAIL}`,
    grantRequest: {},
  });

  // update issue's assignee to first project owner.
  const project = await projectStore.getOrFetchProjectByUID(state.projectId!);
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const ownerList = memberList.filter((member) =>
    member.roleList.includes(PresetRoleType.OWNER)
  );
  const projectOwner = head(ownerList);
  if (projectOwner) {
    newIssue.assignee = `users/${projectOwner.user.email}`;
  }

  const expression: string[] = [];
  const expireDays = state.expireDays;
  expression.push(
    `request.time < timestamp("${dayjs()
      .add(expireDays, "days")
      .toISOString()}")`
  );
  expression.push(`request.row_limit <= ${state.maxRowCount}`);
  if (props.statementOnly) {
    // Selected database condition.
    expression.push(
      stringifyDatabaseResources([
        {
          databaseName: selectedDatabase.value!.name,
        },
      ])
    );
    // Statement condition.
    expression.push(
      `request.statement == "${btoa(
        unescape(encodeURIComponent(state.statement))
      )}"`
    );
  } else {
    if (state.databaseResourceCondition) {
      expression.push(state.databaseResourceCondition);
    }
  }

  const celExpressionString = expression.join(" && ");
  newIssue.grantRequest = {
    role: "roles/EXPORTER",
    user: `users/${currentUser.value.email}`,
    condition: Expr.fromPartial({
      expression: celExpressionString,
    }),
    expiration: Duration.fromPartial({
      seconds: expireDays * 24 * 60 * 60,
    }),
  };

  const createdIssue = await issueServiceClient.createIssue({
    parent: project.name,
    issue: newIssue,
  });

  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("issue.grant-request.request-sent"),
  });

  if (props.redirectToIssuePage) {
    const route = router.resolve({
      path: `/issue/${issueSlug(createdIssue.title, createdIssue.uid)}`,
    });
    window.open(route.href, "_blank");
  }

  emit("close");
};

const generateIssueName = () => {
  if (props.statementOnly) {
    const database = selectedDatabase.value;
    if (!database) {
      throw new Error("Database is not selected");
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
  } else {
    const project = projectStore.getProjectByUID(state.projectId!);
    return `Request data export for "${project.title}"`;
  }
};
</script>
