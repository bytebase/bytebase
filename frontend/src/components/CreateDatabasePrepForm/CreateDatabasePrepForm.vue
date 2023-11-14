<template>
  <div class="space-y-6 divide-y divide-block-border">
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
        <label for="environment" class="textlabel">
          {{ $t("common.environment") }} <span style="color: red">*</span>
        </label>
        <!-- It's default selected to the first env, so we don't need to set `required` here -->
        <EnvironmentSelect
          class="mt-1"
          required
          name="environment"
          :environment="state.environmentId"
          @update:environment="selectEnvironment"
        />
      </div>

      <div class="w-full">
        <div class="flex flex-row items-center space-x-1">
          <InstanceV1EngineIcon
            v-if="state.instanceId"
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
            :instance="state.instanceId"
            :filter="instanceV1HasCreateDatabase"
            @update:instance="selectInstance"
          />
        </div>
      </div>

      <div v-if="requireDatabaseOwnerName" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.database-owner-name") }}
          <span class="text-red-600">*</span>
        </label>
        <InstanceRoleSelect
          class="mt-1"
          name="instance-user"
          :instance-id="state.instanceId"
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

      <div v-if="showAssigneeSelect" class="w-full">
        <label for="user" class="textlabel">
          {{ $t("common.assignee") }} <span class="text-red-600">*</span>
        </label>
        <!-- DBA and Owner always have all access, so we only need to grant to developer -->
        <MemberSelect
          id="user"
          class="mt-1 w-full"
          name="user"
          :allowed-role-list="[UserRole.OWNER, UserRole.DBA]"
          :selected-id="state.assigneeId"
          :placeholder="'Select assignee'"
          @select-user-id="selectAssignee"
        />
      </div>
    </div>
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
import { computed, reactive, PropType } from "vue";
import { useRouter } from "vue-router";
import InstanceRoleSelect from "@/components/InstanceRoleSelect.vue";
import MemberSelect from "@/components/MemberSelect.vue";
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
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import {
  SYSTEM_BOT_ID,
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  UNKNOWN_ID,
  ComposedInstance,
  unknownInstance,
} from "@/types";
import { INTERNAL_RDS_INSTANCE_USER_LIST } from "@/types/InstanceUser";
import { UserRole } from "@/types/proto/v1/auth_service";
import { Engine } from "@/types/proto/v1/common";
import { Backup } from "@/types/proto/v1/database_service";
import { InstanceRole } from "@/types/proto/v1/instance_role_service";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  Plan,
  Plan_CreateDatabaseConfig,
  Plan_Spec,
} from "@/types/proto/v1/rollout_service";
import {
  extractBackupResourceName,
  extractDatabaseResourceName,
  hasWorkspacePermissionV1,
  instanceV1HasCollationAndCharacterSet,
  instanceV1HasCreateDatabase,
} from "@/utils";

interface LocalState {
  projectId?: string;
  environmentId?: string;
  instanceId?: string;
  instanceRole?: string;
  labels: Record<string, string>;
  databaseName: string;
  tableName: string;
  characterSet: string;
  collation: string;
  cluster: string;
  assigneeId?: string;
  showFeatureModal: boolean;
  creating: boolean;
}

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
  environmentId: {
    type: String,
    default: undefined,
  },
  instanceId: {
    type: String,
    default: undefined,
  },
  // If specified, then we are creating a database from the backup.
  backup: {
    type: Object as PropType<Backup>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const instanceV1Store = useInstanceV1Store();
const router = useRouter();

const currentUserV1 = useCurrentUserV1();
const environmentV1Store = useEnvironmentV1Store();
const projectV1Store = useProjectV1Store();

const showAssigneeSelect = computed(() => {
  // If the role can't change assignee after creating the issue, then we will show the
  // assignee select in the prep stage here to request a particular assignee.
  return !hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-issue",
    currentUserV1.value.userRole
  );
});

const state = reactive<LocalState>({
  databaseName: "",
  projectId: props.projectId,
  environmentId: props.environmentId,
  instanceId: props.instanceId,
  labels: {},
  tableName: "",
  characterSet: "",
  collation: "",
  cluster: "",
  assigneeId: showAssigneeSelect.value ? undefined : String(SYSTEM_BOT_ID),
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
    state.environmentId &&
    state.instanceId &&
    state.assigneeId
  );
});

// If project has been specified, then we disallow changing it.
const allowEditProject = computed(() => {
  return !props.projectId;
});

// If instance has been specified, then we disallow changing it.
const allowEditInstance = computed(() => {
  return !props.instanceId;
});

const selectedInstance = computed((): ComposedInstance => {
  return state.instanceId
    ? instanceV1Store.getInstanceByUID(state.instanceId)
    : unknownInstance();
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

const selectEnvironment = (environmentId: string | undefined) => {
  state.environmentId = environmentId;
};

const selectInstance = (instanceId: string | undefined) => {
  state.instanceId = instanceId;
};

const selectInstanceRole = (name?: string) => {
  state.instanceRole = name;
};

const selectAssignee = (assigneeId: string) => {
  state.assigneeId = assigneeId;
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

  const databaseName = state.databaseName;
  const tableName = state.tableName;
  const instanceId = Number(state.instanceId);
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

  const instance = instanceV1Store.getInstanceByUID(String(instanceId));
  const environment = environmentV1Store.getEnvironmentByUID(
    state.environmentId!
  );
  const specs: Plan_Spec[] = [];
  const createDatabaseConfig: Plan_CreateDatabaseConfig = {
    target: instance.name,
    database: databaseName,
    table: tableName,
    labels: state.labels,
    environment: environment.name,

    characterSet:
      state.characterSet ||
      defaultCharsetOfEngineV1(selectedInstance.value.engine),
    collation:
      state.collation ||
      defaultCollationOfEngineV1(selectedInstance.value.engine),
    cluster: state.cluster,
    owner,
    backup: "",
  };
  const spec = Plan_Spec.fromPartial({
    id: uuidv4(),
  });
  specs.push(spec);

  const issueCreate = Issue.fromPartial({
    type: Issue_Type.DATABASE_CHANGE,
    creator: `users/${currentUserV1.value.email}`,
  });

  if (props.backup) {
    spec.restoreDatabaseConfig = {
      backup: props.backup.name,
      createDatabaseConfig,
      // `target` here is the original db
      target: extractDatabaseResourceName(props.backup.name).full,
    };
    const backupTitle = extractBackupResourceName(props.backup.name);
    issueCreate.title = `Create database '${databaseName}' from backup '${backupTitle}'`;
    issueCreate.description = `Creating database '${databaseName}' from backup '${backupTitle}'`;
  } else {
    issueCreate.title = `Create database '${databaseName}'`;
    spec.createDatabaseConfig = createDatabaseConfig;
  }

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
    router.push(`/issue/${createdIssue.uid}`);
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
