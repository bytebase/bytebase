<template>
  <Drawer
    :show="true"
    width="auto"
    :placement="placement"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="
        props.role === PresetRoleType.SQL_EDITOR_USER
          ? $t('custom-approval.risk-rule.risk.namespace.request_query')
          : $t('custom-approval.risk-rule.risk.namespace.request_export')
      "
      :closable="true"
      class="w-[50rem] max-w-[100vw] relative"
    >
      <div class="w-full mx-auto space-y-4">
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-center textlabel mb-2">
            {{ $t("common.databases") }}
            <RequiredStar />
          </span>
          <DatabaseResourceForm
            v-model:database-resources="state.databaseResources"
            :project-name="props.projectName"
            :include-cloumn="false"
            :required-feature="'bb.feature.access-control'"
          />
        </div>
        <div
          v-if="props.role === PresetRoleType.PROJECT_EXPORTER"
          class="w-full flex flex-col justify-start items-start"
        >
          <span class="flex items-center textlabel mb-2">
            {{ $t("issue.grant-request.export-rows") }}
            <RequiredStar />
          </span>
          <MaxRowCountSelect v-model:value="state.maxRowCount" />
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <span class="flex items-start textlabel mb-4">
            {{ $t("common.expiration") }}
            <RequiredStar />
          </span>
          <ExpirationSelector
            v-model:timestamp-in-ms="state.expirationTimestampInMS"
            class="grid-cols-3 sm:grid-cols-4"
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
            <div class="flex items-center gap-1">
              {{ $t("common.ok") }}
            </div>
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { isUndefined } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { PresetRoleType, isValidDatabaseName } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import {
  GrantRequest,
  Issue,
  Issue_Type,
} from "@/types/proto/v1/issue_service";
import { generateIssueTitle } from "@/utils";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import DatabaseResourceForm from "./DatabaseResourceForm/index.vue";
import MaxRowCountSelect from "./MaxRowCountSelect.vue";

interface LocalState {
  databaseResources: DatabaseResource[];
  expirationTimestampInMS?: number;
  description: string;
  maxRowCount: number;
}

const props = withDefaults(
  defineProps<{
    projectName: string;
    role: PresetRoleType.SQL_EDITOR_USER | PresetRoleType.PROJECT_EXPORTER;
    databaseResource?: DatabaseResource;
    placement?: "left" | "right";
  }>(),
  {
    databaseResource: undefined,
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
  const { databaseResource } = props;
  if (
    !databaseResource ||
    !isValidDatabaseName(databaseResource.databaseFullName)
  ) {
    return {
      databaseResources: [],
    };
  }
  return {
    databaseResources: [
      {
        ...databaseResource,
      },
    ],
  };
};

const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  ...extractDatabaseResourcesFromProps(),
  expirationTimestampInMS: undefined,
  description: "",
  maxRowCount: 1000,
});

const allowCreate = computed(() => {
  // If all database selected, the condition is an empty string.
  // If some databases selected, the condition is a string.
  // If no database selected, the condition is undefined.
  if (
    !isUndefined(state.databaseResources) &&
    state.databaseResources.length === 0
  ) {
    return false;
  }
  if (state.expirationTimestampInMS === undefined) {
    return false;
  }
  return true;
});

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const newIssue = Issue.fromPartial({
    title: generateIssueTitle(
      props.role === PresetRoleType.SQL_EDITOR_USER
        ? "bb.issue.grant.request.querier"
        : "bb.issue.grant.request.exporter",
      state.databaseResources?.map(
        (databaseResource) => databaseResource.databaseFullName
      ) ?? []
    ),
    description: state.description,
    type: Issue_Type.GRANT_REQUEST,
    grantRequest: {},
  });

  const project = await useProjectV1Store().getOrFetchProjectByName(
    props.projectName
  );
  const expression: string[] = [];
  if (state.databaseResources) {
    expression.push(stringifyDatabaseResources(state.databaseResources));
  }
  const expirationTimestampInMS = state.expirationTimestampInMS;
  if (expirationTimestampInMS && expirationTimestampInMS > 0) {
    expression.push(
      `request.time < timestamp("${dayjs(
        expirationTimestampInMS
      ).toISOString()}")`
    );
  }
  if (props.role === PresetRoleType.PROJECT_EXPORTER) {
    expression.push(`request.row_limit <= ${state.maxRowCount}`);
  }

  newIssue.grantRequest = GrantRequest.fromPartial({
    role: props.role,
    user: `users/${currentUser.value.email}`,
  });
  if (expression.length > 0) {
    const celExpressionString = expression.join(" && ");
    newIssue.grantRequest.condition = Expr.fromPartial({
      expression: celExpressionString,
    });
  }
  if (expirationTimestampInMS && expirationTimestampInMS > 0) {
    newIssue.grantRequest.expiration = Duration.fromPartial({
      seconds: dayjs(expirationTimestampInMS).unix() - dayjs().unix(),
    });
  }

  const createdIssue = await issueServiceClient.createIssue({
    parent: project.name,
    issue: newIssue,
  });

  const path = `/${createdIssue.name}`;

  window.open(path, "_blank");
};
</script>
