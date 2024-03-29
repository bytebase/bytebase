<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
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
            :filter="filterProject"
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
import { head, isUndefined } from "lodash-es";
import { NButton, NInput, NInputNumber } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { ProjectSelect, DrawerContent, Drawer } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  pushNotification,
} from "@/store";
import type { DatabaseResource, ComposedProject } from "@/types";
import { SYSTEM_BOT_EMAIL, UNKNOWN_ID, PresetRoleType } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Expr } from "@/types/proto/google/type/expr";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  issueSlug,
  memberListInProjectV1,
} from "@/utils";
import DatabaseResourceForm from "../RequestQueryPanel/DatabaseResourceForm/index.vue";

interface LocalState {
  projectId?: string;
  environmentId?: string;
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
  projectId?: string;
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
  if (!state.projectId) {
    return false;
  }
  if (isUndefined(state.databaseResourceCondition)) {
    return false;
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
});

const filterProject = (project: ComposedProject) => {
  return hasProjectPermissionV2(project, currentUser.value, "bb.databases.get");
};

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
    member.roleList.includes(PresetRoleType.PROJECT_OWNER)
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
  if (state.databaseResourceCondition) {
    expression.push(state.databaseResourceCondition);
  }

  const celExpressionString = expression.join(" && ");
  newIssue.grantRequest = {
    role: PresetRoleType.PROJECT_EXPORTER,
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
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.name),
        issueSlug: issueSlug(createdIssue.title, createdIssue.uid),
      },
    });
    window.open(route.href, "_blank");
  }

  emit("close");
};

const generateIssueName = () => {
  const project = projectStore.getProjectByUID(state.projectId!);
  return `Request data export for "${project.title}"`;
};
</script>
