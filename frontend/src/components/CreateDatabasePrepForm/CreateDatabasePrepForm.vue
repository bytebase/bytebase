<template>
  <div class="w-72 mx-auto space-y-4">
    <div class="w-full">
      <label for="project" class="textlabel">
        {{ $t("common.project") }} <span style="color: red">*</span>
      </label>
      <ProjectSelect
        class="mt-1 !w-full"
        required
        :disabled="!allowEditProject"
        :project="state.projectId"
        @update:project="selectProject"
      />
    </div>

    <div class="w-full">
      <label for="name" class="textlabel">
        {{ $t("create-db.new-database-name") }}
        <span class="text-red-600">*</span>
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
        <span class="text-red-600">*</span>
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
      <div class="flex flex-row items-center space-x-1">
        <InstanceV1EngineIcon
          v-if="state.instance"
          :instance="selectedInstance"
        />
        <label for="instance" class="textlabel">
          {{ $t("common.instance") }} <span class="text-red-600">*</span>
        </label>
      </div>
      <div class="flex flex-row space-x-2 items-center">
        <InstanceSelect
          class="mt-1"
          name="instance"
          required
          :disabled="!allowEditInstance"
          :instance="state.instance"
          :use-resource-id="true"
          :filter="instanceV1HasCreateDatabase"
          @update:instance="selectInstance"
        />
      </div>
    </div>

    <div class="w-full">
      <label for="environment" class="textlabel">
        {{ $t("common.environment") }}
      </label>
      <EnvironmentSelect
        v-model:environment="state.environment"
        class="mt-1"
        required
        name="environment"
        :use-resource-id="true"
      />
    </div>

    <div v-if="requireDatabaseOwnerName" class="w-full">
      <label for="name" class="textlabel">
        {{ $t("create-db.database-owner-name") }}
        <span class="text-red-600">*</span>
      </label>
      <InstanceRoleSelect
        class="mt-1"
        name="instance-user"
        :instance="state.instance"
        :role="state.instanceRole"
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

  <FeatureModal
    feature="bb.feature.multi-tenancy"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
  <div
    v-if="state.creating"
    class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
  >
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { isEmpty } from "lodash-es";
import { NInput } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import InstanceRoleSelect from "@/components/InstanceRoleSelect.vue";
import {
  ProjectSelect,
  EnvironmentSelect,
  InstanceSelect,
  InstanceV1EngineIcon,
} from "@/components/v2";
import {
  experimentalCreateIssueByPlan,
  hasFeature,
  useCurrentUserV1,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import type { ComposedInstance } from "@/types";
import {
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  UNKNOWN_ID,
} from "@/types";
import { INTERNAL_RDS_INSTANCE_USER_LIST } from "@/types/InstanceUser";
import { Engine } from "@/types/proto/v1/common";
import type { InstanceRole } from "@/types/proto/v1/instance_role_service";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import type { Plan_CreateDatabaseConfig } from "@/types/proto/v1/plan_service";
import { Plan, Plan_Spec } from "@/types/proto/v1/plan_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  instanceV1HasCollationAndCharacterSet,
  instanceV1HasCreateDatabase,
} from "@/utils";

interface LocalState {
  projectId?: string;
  environment?: string;
  instance?: string;
  instanceRole?: string;
  labels: Record<string, string>;
  databaseName: string;
  tableName: string;
  characterSet: string;
  collation: string;
  cluster: string;
  showFeatureModal: boolean;
  creating: boolean;
}

const props = defineProps<{
  projectId?: string;
  environment?: string;
  instance?: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const instanceV1Store = useInstanceV1Store();
const router = useRouter();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  databaseName: "",
  projectId: props.projectId,
  environment: props.environment,
  instance: props.instance,
  labels: {},
  tableName: "",
  characterSet: "",
  collation: "",
  cluster: "",
  showFeatureModal: false,
  creating: false,
});

const project = computed(() => {
  return projectV1Store.getProjectByUID(state.projectId ?? String(UNKNOWN_ID));
});

const isReservedName = computed(() => {
  return state.databaseName.toLowerCase() == "bytebase";
});

const isTenantProject = computed((): boolean => {
  if (parseInt(project.value.uid, 10) === UNKNOWN_ID) {
    return false;
  }

  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const allowCreate = computed(() => {
  return (
    !isEmpty(state.databaseName) &&
    validDatabaseOwnerName.value &&
    !isReservedName.value &&
    state.projectId &&
    state.environment &&
    state.instance
  );
});

// If project has been specified, then we disallow changing it.
const allowEditProject = computed(() => {
  return !props.projectId;
});

// If instance has been specified, then we disallow changing it.
const allowEditInstance = computed(() => {
  return !props.instance;
});

const selectedInstance = computed((): ComposedInstance => {
  return instanceV1Store.getInstanceByName(state.instance ?? "");
});

const showCollationAndCharacterSet = computed((): boolean => {
  const instance = selectedInstance.value;
  return instanceV1HasCollationAndCharacterSet(instance);
});

const requireDatabaseOwnerName = computed((): boolean => {
  const instance = selectedInstance.value;
  if (instance.uid === String(UNKNOWN_ID)) {
    return false;
  }
  return [Engine.POSTGRES, Engine.REDSHIFT].includes(instance.engine);
});

const validDatabaseOwnerName = computed((): boolean => {
  if (!requireDatabaseOwnerName.value) {
    return true;
  }

  return state.instanceRole !== undefined;
});

const selectProject = (projectId: string | undefined) => {
  state.projectId = projectId;
};

const selectInstance = (instance: string | undefined) => {
  state.instance = instance;
  state.environment = selectedInstance.value.environment;
};

const selectInstanceRole = (name?: string) => {
  state.instanceRole = name;
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
  if (!state.environment || !state.instance) {
    return;
  }

  const databaseName = state.databaseName;
  const tableName = state.tableName;

  let owner = "";
  if (requireDatabaseOwnerName.value && state.instanceRole) {
    const instanceUser = await instanceV1Store.fetchInstanceRoleByName(
      state.instanceRole
    );
    owner = instanceUser.roleName;
  }

  if (isTenantProject.value) {
    if (!hasFeature("bb.feature.multi-tenancy")) {
      state.showFeatureModal = true;
      return;
    }
  }

  const specs: Plan_Spec[] = [];
  const createDatabaseConfig: Plan_CreateDatabaseConfig = {
    target: state.instance,
    database: databaseName,
    table: tableName,
    labels: state.labels,
    environment: state.environment,

    characterSet:
      state.characterSet ||
      defaultCharsetOfEngineV1(selectedInstance.value.engine),
    collation:
      state.collation ||
      defaultCollationOfEngineV1(selectedInstance.value.engine),
    cluster: state.cluster,
    owner,
  };
  const spec = Plan_Spec.fromPartial({
    id: uuidv4(),
  });
  specs.push(spec);

  const issueCreate = Issue.fromPartial({
    type: Issue_Type.DATABASE_CHANGE,
    creator: `users/${currentUserV1.value.email}`,
  });

  issueCreate.title = `Create database '${databaseName}'`;
  spec.createDatabaseConfig = createDatabaseConfig;

  state.creating = true;
  try {
    const planCreate = Plan.fromJSON({
      steps: [{ specs: [spec] }],
      creator: currentUserV1.value.name,
    });
    const { createdIssue } = await experimentalCreateIssueByPlan(
      project.value,
      issueCreate,
      planCreate
    );
    router.push({
      path: `/${createdIssue.name}`,
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
