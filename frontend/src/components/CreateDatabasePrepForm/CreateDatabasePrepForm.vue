<template>
  <div class="w-72 mx-auto space-y-4">
    <div v-if="allowEditProject" class="w-full">
      <label for="project" class="textlabel">
        {{ $t("common.project") }}
        <RequiredStar />
      </label>
      <ProjectSelect
        class="mt-1 !w-full"
        required
        :project-name="state.projectName"
        @update:project-name="selectProject"
      />
    </div>

    <div class="w-full">
      <div class="flex flex-row items-center space-x-1">
        <label for="instance" class="textlabel">
          {{ $t("common.instance") }} <RequiredStar />
        </label>
      </div>
      <div class="flex flex-row space-x-2 items-center">
        <InstanceSelect
          class="mt-1"
          name="instance"
          required
          :disabled="!allowEditInstance"
          :instance-name="state.instanceName"
          :allowed-engine-list="supportedEngines"
          @update:instance-name="selectInstance"
        />
      </div>
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
        <RequiredStar />
      </label>
      <EnvironmentSelect
        v-model:environment-name="state.environmentName"
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
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import { NInput } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import InstanceRoleSelect from "@/components/InstanceRoleSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  EnvironmentSelect,
  InstanceSelect,
  ProjectSelect,
} from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  experimentalCreateIssueByPlan,
  useCurrentUserV1,
  useInstanceResourceByName,
  useProjectByName,
} from "@/store";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  isValidEnvironmentName,
  isValidInstanceName,
  isValidProjectName,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { IssueSchema, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_CreateDatabaseConfig } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanSchema,
  Plan_SpecSchema,
  Plan_CreateDatabaseConfigSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  enginesSupportCreateDatabase,
  extractProjectResourceName,
  instanceV1HasCollationAndCharacterSet,
  issueV1Slug,
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
  projectName?: string;
  environmentName?: string;
  instanceName?: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  databaseName: "",
  projectName: props.projectName,
  environmentName: props.environmentName,
  instanceName: props.instanceName,
  tableName: "",
  characterSet: "",
  collation: "",
  cluster: "",
  creating: false,
});
const { project } = useProjectByName(
  computed(() => state.projectName ?? UNKNOWN_PROJECT_NAME)
);

const isReservedName = computed(() => {
  return state.databaseName.toLowerCase() == "bytebase";
});

const supportedEngines = computed(() => enginesSupportCreateDatabase());

const allowCreate = computed(() => {
  return (
    !isEmpty(state.databaseName) &&
    validDatabaseOwnerName.value &&
    !isReservedName.value &&
    isValidProjectName(state.projectName) &&
    isValidEnvironmentName(state.environmentName) &&
    isValidInstanceName(state.instanceName)
  );
});

// If project has been specified, then we disallow changing it.
const allowEditProject = computed(() => {
  return !props.projectName;
});

// If instance has been specified, then we disallow changing it.
const allowEditInstance = computed(() => {
  return !props.instanceName;
});

const selectedInstance = computed(
  () => useInstanceResourceByName(state.instanceName ?? "").instance.value
);

const showCollationAndCharacterSet = computed((): boolean => {
  const instance = selectedInstance.value;
  return instanceV1HasCollationAndCharacterSet(instance);
});

const requireDatabaseOwnerName = computed((): boolean => {
  const instance = selectedInstance.value;
  if (!isValidInstanceName(instance.name)) {
    return false;
  }
  return [Engine.POSTGRES, Engine.REDSHIFT, Engine.COCKROACHDB].includes(
    instance.engine
  );
});

const validDatabaseOwnerName = computed((): boolean => {
  if (!requireDatabaseOwnerName.value) {
    return true;
  }

  return state.instanceRole !== undefined;
});

const selectProject = (name: string | undefined) => {
  state.projectName = name;
};

const selectInstance = (instanceName: string | undefined) => {
  state.instanceName = instanceName;
  state.environmentName = selectedInstance.value.environment;
};

const selectInstanceRole = (role?: InstanceRole) => {
  state.instanceRole = role;
};

const filterInstanceRole = (user: InstanceRole) => {
  if (INTERNAL_RDS_INSTANCE_USER_LIST.includes(user.roleName)) {
    return false;
  }
  return true;
};

const cancel = () => {
  emit("dismiss");
};

const createV1 = async () => {
  if (!allowCreate.value) {
    return;
  }
  if (!state.environmentName || !state.instanceName) {
    return;
  }

  const databaseName = state.databaseName;
  const tableName = state.tableName;

  let owner = "";
  if (requireDatabaseOwnerName.value && state.instanceRole) {
    owner = state.instanceRole.roleName;
  }

  const specs: Plan_Spec[] = [];
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
  });
  specs.push(spec);

  const issueCreate = createProto(IssueSchema, {
    type: Issue_Type.DATABASE_CHANGE,
    creator: `users/${currentUserV1.value.email}`,
  });

  spec.config = {
    case: "createDatabaseConfig",
    value: createDatabaseConfig,
  };

  const title = `${t("issue.title.create-database")} '${databaseName}'`;
  state.creating = true;
  try {
    const planCreate = createProto(PlanSchema, {
      title: title,
      specs: [spec],
      creator: currentUserV1.value.name,
    });
    const { createdIssue } = await experimentalCreateIssueByPlan(
      project.value,
      issueCreate,
      planCreate
    );
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(createdIssue.name),
        issueSlug: issueV1Slug(createdIssue.name, createdIssue.title),
      },
    });
  } finally {
    state.creating = false;
  }
};

const create = async () => {
  await createV1();
};

defineExpose({
  allowCreate,
  cancel,
  create,
});
</script>
