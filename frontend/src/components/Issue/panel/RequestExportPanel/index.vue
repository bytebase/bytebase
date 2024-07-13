<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('custom-approval.risk-rule.risk.namespace.request_export')"
      :closable="true"
      class="w-[50rem] max-w-[100vw] relative"
    >
      <div class="w-full mx-auto space-y-4">
        <div class="w-full flex flex-row justify-start items-center gap-2">
          <span class="flex items-center textlabel">
            {{ $t("common.project") }}
            <RequiredStar />
          </span>
          <ProjectSelect
            class="!w-60 shrink-0"
            :project-name="state.projectName"
            :filter="filterProject"
            :disabled="isValidProjectName(props.projectName)"
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
            class="grid-cols-3 sm:grid-cols-4 md:grid-cols-6"
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
import { isUndefined } from "lodash-es";
import { NButton, NInput, NInputNumber } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { ProjectSelect, DrawerContent, Drawer } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  pushNotification,
} from "@/store";
import type { DatabaseResource, ComposedProject } from "@/types";
import { UNKNOWN_ID, PresetRoleType, isValidProjectName } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import {
  GrantRequest,
  Issue,
  Issue_Type,
} from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";
import DatabaseResourceForm from "../RequestQueryPanel/DatabaseResourceForm/index.vue";

interface LocalState {
  projectName?: string;
  environmentName?: string;
  databaseId?: string;
  databaseResourceCondition?: string;
  databaseResources: DatabaseResource[];
  expireDays: number;
  maxRowCount: number;
  description: string;
}

defineOptions({
  name: "RequestExportPanel",
});

const props = defineProps<{
  projectName?: string;
  databaseId?: string;
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
  description: "",
});

const allowCreate = computed(() => {
  if (!state.projectName) {
    return false;
  }
  if (isUndefined(state.databaseResourceCondition)) {
    return false;
  }
  return true;
});

onMounted(async () => {
  if (props.projectName) {
    handleProjectSelect(props.projectName);
  }
  if (props.databaseId) {
    handleDatabaseSelect(props.databaseId);
  }
});

const filterProject = (project: ComposedProject) => {
  return hasProjectPermissionV2(project, currentUser.value, "bb.databases.get");
};

const handleProjectSelect = async (name: string | undefined) => {
  state.projectName = name;
};

const handleEnvironmentSelect = (environmentName: string | undefined) => {
  state.environmentName = environmentName;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  // Unselect database if it doesn't belong to the newly selected environment.
  if (
    database &&
    database.uid !== String(UNKNOWN_ID) &&
    database.effectiveEnvironment !== state.environmentName
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
    handleProjectSelect(database.project);
    handleEnvironmentSelect(database.effectiveEnvironment);
  }
};

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const project = await projectStore.getOrFetchProjectByName(
    state.projectName!
  );
  const newIssue = Issue.fromPartial({
    title: generateIssueName(project),
    description: state.description,
    type: Issue_Type.GRANT_REQUEST,
    grantRequest: {},
  });

  const expression: string[] = [];
  expression.push(`request.row_limit <= ${state.maxRowCount}`);
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
    role: PresetRoleType.PROJECT_EXPORTER,
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

  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("issue.grant-request.request-sent"),
  });

  if (props.redirectToIssuePage) {
    const route = router.resolve({
      path: `/${createdIssue.name}`,
    });
    window.open(route.href, "_blank");
  }

  emit("close");
};

const generateIssueName = (project: ComposedProject) => {
  return `Request data export for "${project.title}"`;
};
</script>
