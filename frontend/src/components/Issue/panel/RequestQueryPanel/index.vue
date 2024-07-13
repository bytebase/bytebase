<template>
  <Drawer
    :show="true"
    width="auto"
    :placement="placement"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('custom-approval.risk-rule.risk.namespace.request_query')"
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
            :project-name="state.projectName"
            :allowed-project-role-list="[
              PresetRoleType.PROJECT_OWNER,
              PresetRoleType.PROJECT_DEVELOPER,
              PresetRoleType.PROJECT_VIEWER,
            ]"
            @update:project-name="handleProjectSelect"
          />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("common.databases") }}
            <RequiredStar />
          </span>
          <DatabaseResourceForm
            :project-name="state.projectName"
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
            class="grid-cols-3 sm:grid-cols-4"
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
import { isUndefined, uniq } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
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
import type { ComposedDatabase, DatabaseResource } from "@/types";
import { UNKNOWN_ID, PresetRoleType } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import {
  GrantRequest,
  Issue,
  Issue_Type,
} from "@/types/proto/v1/issue_service";
import DatabaseResourceForm from "./DatabaseResourceForm/index.vue";

interface LocalState {
  projectName?: string;
  databaseResourceCondition?: string;
  databaseResources: DatabaseResource[];
  expireDays: number;
  description: string;
}

defineOptions({
  name: "RequestQueryPanel",
});

const props = withDefaults(
  defineProps<{
    projectName?: string;
    database?: ComposedDatabase;
    placement: "left" | "right";
  }>(),
  {
    projectName: undefined,
    database: undefined,
    placement: "right",
  }
);

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

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  projectName: props.projectName,
  ...extractDatabaseResourcesFromProps(),
  expireDays: 1,
  description: "",
});

const allowCreate = computed(() => {
  if (!state.projectName) {
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

const handleProjectSelect = async (name: string | undefined) => {
  state.projectName = name;
};

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const newIssue = Issue.fromPartial({
    title: generateIssueName(),
    description: state.description,
    type: Issue_Type.GRANT_REQUEST,
    grantRequest: {},
  });

  const project = await useProjectV1Store().getOrFetchProjectByName(
    state.projectName!
  );
  const expression: string[] = [];
  if (state.databaseResourceCondition) {
    expression.push(state.databaseResourceCondition);
  }
  const expireDays = state.expireDays;
  if (expireDays > 0) {
    expression.push(
      `request.time < timestamp("${dayjs()
        .add(expireDays, "days")
        .toISOString()}")`
    );
  }

  newIssue.grantRequest = GrantRequest.fromPartial({
    role: PresetRoleType.PROJECT_QUERIER,
    user: `users/${currentUser.value.email}`,
  });
  if (expression.length > 0) {
    const celExpressionString = expression.join(" && ");
    newIssue.grantRequest.condition = Expr.fromPartial({
      expression: celExpressionString,
    });
  }
  if (expireDays > 0) {
    newIssue.grantRequest.expiration = Duration.fromPartial({
      seconds: expireDays * 24 * 60 * 60,
    });
  }

  const createdIssue = await issueServiceClient.createIssue({
    parent: project.name,
    issue: newIssue,
  });

  router.push({
    path: `/${createdIssue.name}`,
  });
};

const generateIssueName = () => {
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
