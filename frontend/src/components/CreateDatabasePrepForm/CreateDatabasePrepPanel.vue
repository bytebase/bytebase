<template>
  <DrawerContent
    :title="$t('quick-action.create-db')"
    class="w-[40rem] max-w-[100vw]"
  >
    <div class="mx-auto flex flex-col gap-y-4">
      <div class="w-full">
        <label for="project" class="textlabel">
          {{ $t("common.project") }}
          <RequiredStar />
        </label>
        <ProjectSelect
          class="mt-1 w-full!"
          required
          :disabled="isValidProjectName(currentProject.name)"
          v-model:value="state.projectName"
        />
      </div>

      <div class="w-full">
        <label for="instance" class="textlabel">
          {{ $t("common.instance") }} <RequiredStar />
        </label>
        <InstanceSelect
          class="mt-1"
          name="instance"
          required
          :disabled="!allowEditInstance"
          :value="state.instanceName"
          :allowed-engine-list="supportedEngines"
          @update:value="selectInstance"
        />
      </div>

      <div class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.new-database-name") }}
          <RequiredStar />
        </label>
        <NInput
          v-model:value="state.databaseName"
          required
          name="databaseName"
          type="text"
          class="mt-1 w-full"
          :placeholder="$t('create-db.new-database-name')"
        />
        <span v-if="isReservedName" class="text-red-600">
          <i18n-t keypath="create-db.reserved-db-error">
            <template #databaseName>
              {{ state.databaseName }}
            </template>
          </i18n-t>
        </span>
      </div>

      <div v-if="selectedInstance.engine === Engine.MONGODB" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.new-collection-name") }}
          <RequiredStar />
        </label>
        <NInput
          v-model:value="state.tableName"
          required
          name="tableName"
          type="text"
          class="mt-1 w-full"
        />
      </div>

      <div v-if="selectedInstance.engine === Engine.CLICKHOUSE" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.cluster") }}
        </label>
        <NInput
          v-model:value="state.cluster"
          name="cluster"
          type="text"
          class="mt-1 w-full"
        />
      </div>

      <div class="w-full">
        <label for="environment" class="textlabel">
          {{ $t("common.environment") }}
        </label>
        <EnvironmentSelect
          v-model:value="state.environmentName"
          class="mt-1"
          required
          name="environment"
        />
      </div>

      <div v-if="requireDatabaseOwnerName && state.instanceName" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.database-owner-name") }}
          <RequiredStar />
        </label>
        <InstanceRoleSelect
          class="mt-1"
          name="instance-user"
          :instance-name="state.instanceName"
          :role="state.instanceRole?.name"
          :filter="filterInstanceRole"
          @update:instance-role="selectInstanceRole"
        />
      </div>

      <template v-if="showCollationAndCharacterSet">
        <div class="w-full">
          <label for="charset" class="textlabel">
            {{
              selectedInstance.engine === Engine.POSTGRES ||
              selectedInstance.engine === Engine.REDSHIFT
                ? $t("db.encoding")
                : $t("db.character-set")
            }}</label
          >
          <NInput
            v-model:value="state.characterSet"
            name="charset"
            type="text"
            class="mt-1 w-full"
            :placeholder="defaultCharsetOfEngineV1(selectedInstance.engine)"
          />
        </div>

        <div class="w-full">
          <label for="collation" class="textlabel">
            {{ $t("db.collation") }}
          </label>
          <NInput
            v-model:value="state.collation"
            name="collation"
            type="text"
            class="mt-1 w-full"
            :placeholder="
              defaultCollationOfEngineV1(selectedInstance.engine) || 'default'
            "
          />
        </div>
      </template>
    </div>

    <div
      v-if="state.creating"
      class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
    >
      <BBSpin />
    </div>
    <template #footer>
      <div class="flex justify-end gap-x-3">
        <NButton quaternary @click.prevent="cancel">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowCreate"
          @click.prevent="create"
        >
          {{ $t("common.create") }}
        </NButton>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import InstanceRoleSelect from "@/components/InstanceRoleSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  DrawerContent,
  EnvironmentSelect,
  InstanceSelect,
  ProjectSelect,
} from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import {
  experimentalCreateIssueByPlan,
  useCurrentProjectV1,
  useCurrentUserV1,
  useInstanceResourceByName,
  useProjectV1Store,
} from "@/store";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { Issue_Type, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_CreateDatabaseConfig } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  enginesSupportCreateDatabase,
  extractIssueUID,
  extractProjectResourceName,
  instanceV1HasCollationAndCharacterSet,
} from "@/utils";

const INTERNAL_RDS_INSTANCE_USER_LIST = ["rds_ad", "rdsadmin", "rds_iam"];

interface LocalState {
  projectName?: string;
  environmentName?: string;
  instanceName?: string;
  instanceRole?: InstanceRole;
  databaseName: string;
  tableName: string;
  characterSet: string;
  collation: string;
  cluster: string;
  creating: boolean;
}

const props = defineProps<{
  environmentName?: string;
  instanceName?: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const projectStore = useProjectV1Store();
const { project: currentProject } = useCurrentProjectV1();

const state = reactive<LocalState>({
  databaseName: "",
  projectName: isValidProjectName(currentProject.value.name)
    ? currentProject.value.name
    : undefined,
  environmentName: props.environmentName,
  instanceName: props.instanceName,
  tableName: "",
  characterSet: "",
  collation: "",
  cluster: "",
  creating: false,
});

const isReservedName = computed(() => {
  return state.databaseName.toLowerCase() === "bytebase";
});

const supportedEngines = computed(() => enginesSupportCreateDatabase());

const allowCreate = computed(() => {
  return (
    isValidProjectName(state.projectName) &&
    isValidInstanceName(state.instanceName) &&
    !isEmpty(state.databaseName) &&
    validDatabaseOwnerName.value &&
    !isReservedName.value
  );
});

// If instance has been specified, then we disallow changing it.
const allowEditInstance = computed(() => !props.instanceName);

const selectedInstance = computed(
  () => useInstanceResourceByName(state.instanceName ?? "").instance.value
);

const showCollationAndCharacterSet = computed(() =>
  instanceV1HasCollationAndCharacterSet(selectedInstance.value)
);

const requireDatabaseOwnerName = computed(() => {
  const instance = selectedInstance.value;
  return (
    isValidInstanceName(instance.name) &&
    [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
      instance.engine
    )
  );
});

const validDatabaseOwnerName = computed(
  () => !requireDatabaseOwnerName.value || state.instanceRole !== undefined
);

const selectInstance = (instanceName: string | undefined) => {
  state.instanceName = instanceName;
  state.environmentName = selectedInstance.value.environment;
};

const selectInstanceRole = (role?: InstanceRole) => {
  state.instanceRole = role;
};

const filterInstanceRole = (user: InstanceRole) =>
  !INTERNAL_RDS_INSTANCE_USER_LIST.includes(user.roleName);

const cancel = () => {
  emit("dismiss");
};

const create = async () => {
  if (!allowCreate.value) {
    return;
  }

  const { databaseName, tableName } = state;
  const owner =
    requireDatabaseOwnerName.value && state.instanceRole
      ? state.instanceRole.roleName
      : "";

  const createDatabaseConfig: Plan_CreateDatabaseConfig = createProto(
    Plan_CreateDatabaseConfigSchema,
    {
      target: state.instanceName,
      database: databaseName,
      table: tableName,
      environment: state.environmentName,

      characterSet:
        state.characterSet ||
        defaultCharsetOfEngineV1(selectedInstance.value.engine),
      collation:
        state.collation ||
        defaultCollationOfEngineV1(selectedInstance.value.engine),
      cluster: state.cluster,
      owner,
    }
  );
  const spec = createProto(Plan_SpecSchema, {
    id: uuidv4(),
    config: {
      case: "createDatabaseConfig",
      value: createDatabaseConfig,
    },
  });

  const title = `${t("issue.title.create-database")} '${databaseName}'`;
  state.creating = true;
  try {
    const planCreate = createProto(PlanSchema, {
      title,
      specs: [spec],
      creator: currentUserV1.value.name,
    });
    const issueCreate = createProto(IssueSchema, {
      type: Issue_Type.DATABASE_CHANGE,
      creator: `users/${currentUserV1.value.email}`,
    });
    const project = await projectStore.getOrFetchProjectByName(
      state.projectName!
    );
    const { createdIssue } = await experimentalCreateIssueByPlan(
      project,
      issueCreate,
      planCreate,
      { skipRollout: true }
    );
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId: extractProjectResourceName(createdIssue.name),
        issueId: extractIssueUID(createdIssue.name),
      },
    });
  } finally {
    state.creating = false;
  }
};
</script>
