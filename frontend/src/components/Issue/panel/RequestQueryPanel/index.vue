<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('quick-action.request-query-permission')"
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
            :filter-by-current-user="true"
            :allowed-project-role-list="[
              PresetRoleType.OWNER,
              PresetRoleType.DEVELOPER,
              PresetRoleType.VIEWER,
            ]"
            @update:project="handleProjectSelect"
          />
        </div>
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
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-start textlabel mb-4">
            {{ $t("common.expiration") }}
            <RequiredStar />
          </span>
          <ExpirationSelector
            class="grid-cols-4"
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
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head, isUndefined, uniq } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent, ProjectSelect } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import {
  ComposedDatabase,
  DatabaseResource,
  PresetRoleType,
  SYSTEM_BOT_EMAIL,
  UNKNOWN_ID,
} from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import { issueSlug, memberListInProjectV1 } from "@/utils";
import DatabaseResourceForm from "./DatabaseResourceForm/index.vue";

interface LocalState {
  projectId?: string;
  databaseResourceCondition?: string;
  databaseResources: DatabaseResource[];
  expireDays: number;
  description: string;
}

const props = defineProps<{
  projectId?: string;
  database?: ComposedDatabase;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const extractDatabaseResourcesFromProps = (): Pick<
  LocalState,
  "databaseResources"
> => {
  const { database } = props;
  if (!database || database.uid === String(UNKNOWN_ID)) {
    return {
      databaseResources: [],
    };
  }
  return {
    databaseResources: [
      {
        databaseName: database.name,
      },
    ],
  };
};

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  projectId: props.projectId,
  ...extractDatabaseResourcesFromProps(),
  expireDays: 7,
  description: "",
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

const allowCreate = computed(() => {
  if (!state.projectId) {
    return false;
  }

  // If all database selected, the condition is an empty string.
  // If some databases selected, the condition is a string.
  // If no database selected, the condition is undefined.
  if (isUndefined(state.databaseResourceCondition)) {
    return false;
  }
  return true;
});

const handleProjectSelect = async (projectId: string | undefined) => {
  state.projectId = projectId;
};

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  // const newIssue: IssueCreate = {
  //   name: generateIssueName(),
  //   type: "bb.issue.grant.request",
  //   description: state.description,
  //   projectId: Number(state.projectId),
  //   assigneeId: SYSTEM_BOT_ID,
  //   createContext: {},
  //   payload: {},
  // };
  const newIssue = Issue.fromPartial({
    title: generateIssueName(),
    description: state.description,
    type: Issue_Type.GRANT_REQUEST,
    assignee: `users/${SYSTEM_BOT_EMAIL}`,
    grantRequest: {},
  });

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
    newIssue.assignee = `users/${projectOwner.user.email}`;
  }

  const expression: string[] = [];
  const expireDays = state.expireDays;
  expression.push(
    `request.time < timestamp("${dayjs()
      .add(expireDays, "days")
      .toISOString()}")`
  );
  if (state.databaseResourceCondition) {
    expression.push(state.databaseResourceCondition);
  }

  const celExpressionString = expression.join(" && ");
  newIssue.grantRequest = {
    role: "roles/QUERIER",
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

  router.push(`/issue/${issueSlug(createdIssue.title, createdIssue.uid)}`);
};

const generateIssueName = () => {
  if (!state.projectId) {
    throw new Error("No project selected");
  }

  if (state.databaseResources.length === 0) {
    return `Request query for all database`;
  } else {
    const databaseNames = uniq(
      state.databaseResources.map(
        (databaseResource) => databaseResource.databaseName
      )
    );
    const databases = databaseNames.map((name) =>
      databaseStore.getDatabaseByName(name)
    );
    return `Request query for "${databases
      .map((database) => database.databaseName)
      .join(", ")}"`;
  }
};
</script>
