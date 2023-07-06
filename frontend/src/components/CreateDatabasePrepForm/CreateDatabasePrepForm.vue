<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="w-72 mx-auto space-y-4">
      <div class="w-full">
        <label for="project" class="textlabel">
          {{ $t("common.project") }} <span style="color: red">*</span>
        </label>
        <ProjectSelect
          id="project"
          class="mt-1"
          name="project"
          required
          :disabled="!allowEditProject"
          :selected-id="state.projectId"
          @select-project-id="selectProject"
        />
      </div>

      <div class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.new-database-name") }}
          <span class="text-red-600">*</span>
        </label>
        <input
          id="databaseName"
          v-model="state.databaseName"
          required
          name="databaseName"
          type="text"
          class="textfield mt-1 w-full"
        />
        <span v-if="isReservedName" class="text-red-600">
          <i18n-t keypath="create-db.reserved-db-error">
            <template #databaseName>
              {{ state.databaseName }}
            </template>
          </i18n-t>
        </span>
        <DatabaseNameTemplateTips
          v-if="isDbNameTemplateMode"
          :project="project"
          :name="state.databaseName"
          :labels="state.labels"
        />
      </div>

      <div v-if="selectedInstance.engine === Engine.MONGODB" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.new-collection-name") }}
          <span class="text-red-600">*</span>
        </label>
        <input
          id="tableName"
          v-model="state.tableName"
          required
          name="tableName"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <div v-if="selectedInstance.engine === Engine.CLICKHOUSE" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.cluster") }}
        </label>
        <input
          id="name"
          v-model="state.cluster"
          name="cluster"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <!-- Providing more dropdowns for required labels as if they are normal required props of DB -->
      <DatabaseLabelForm
        v-if="isTenantProject"
        ref="labelForm"
        :project="project"
        :labels="state.labels"
        filter="required"
      />

      <div class="w-full">
        <label for="environment" class="textlabel">
          {{ $t("common.environment") }} <span style="color: red">*</span>
        </label>
        <!-- It's default selected to the first env, so we don't need to set `required` here -->
        <EnvironmentSelect
          id="environment"
          class="mt-1 w-full"
          name="environment"
          :disabled="!allowEditEnvironment"
          :selected-id="state.environmentId"
          @select-environment-id="selectEnvironment"
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
            id="instance"
            class="mt-1"
            name="instance"
            required
            :disabled="!allowEditInstance"
            :selected-id="state.instanceId"
            :environment-id="state.environmentId"
            :filter="instanceV1HasCreateDatabase"
            @select-instance-id="selectInstance"
          />
        </div>
      </div>

      <div v-if="requireDatabaseOwnerName" class="w-full">
        <label for="name" class="textlabel">
          {{ $t("create-db.database-owner-name") }}
          <span class="text-red-600">*</span>
        </label>
        <InstanceRoleSelect
          id="instance-user"
          class="mt-1"
          name="instance-user"
          :instance-id="state.instanceId"
          :role="state.instanceRole"
          :filter="filterInstanceRole"
          @select="selectInstanceRole"
        />
      </div>

      <!-- Providing other dropdowns for optional labels as if they are normal optional props of DB -->
      <DatabaseLabelForm
        v-if="isTenantProject"
        class="w-full"
        :project="project"
        :labels="state.labels"
        filter="optional"
      />

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
          <input
            id="charset"
            v-model="state.characterSet"
            name="charset"
            type="text"
            class="textfield mt-1 w-full"
            :placeholder="defaultCharsetOfEngineV1(selectedInstance.engine)"
          />
        </div>

        <div class="w-full">
          <label for="collation" class="textlabel">
            {{ $t("db.collation") }}
          </label>
          <input
            id="collation"
            v-model="state.collation"
            name="collation"
            type="text"
            class="textfield mt-1 w-full"
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
import { computed, reactive, PropType, watchEffect, ref, toRef } from "vue";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash-es";
import { useEventListener } from "@vueuse/core";

import { InstanceV1EngineIcon } from "@/components/v2";
import {
  DatabaseLabelForm,
  DatabaseNameTemplateTips,
  useDBNameTemplateInputState,
} from "./";
import InstanceSelect from "@/components/InstanceSelect.vue";
import EnvironmentSelect from "@/components/EnvironmentSelect.vue";
import ProjectSelect from "@/components/ProjectSelect.vue";
import MemberSelect from "@/components/MemberSelect.vue";
import InstanceRoleSelect from "@/components/InstanceRoleSelect.vue";
import {
  IssueCreate,
  SYSTEM_BOT_ID,
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
  CreateDatabaseContext,
  UNKNOWN_ID,
  PITRContext,
  ComposedInstance,
  unknownInstance,
} from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { INTERNAL_RDS_INSTANCE_USER_LIST } from "@/types/InstanceUser";
import {
  extractDatabaseResourceName,
  extractEnvironmentResourceName,
  hasWorkspacePermissionV1,
  instanceV1HasCollationAndCharacterSet,
  instanceV1HasCreateDatabase,
  issueSlug,
} from "@/utils";
import {
  hasFeature,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useIssueStore,
  useProjectV1Store,
} from "@/store";
import { UserRole } from "@/types/proto/v1/auth_service";
import { Engine } from "@/types/proto/v1/common";
import { InstanceRole } from "@/types/proto/v1/instance_role_service";
import { Backup } from "@/types/proto/v1/database_service";

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
const projectV1Store = useProjectV1Store();

useEventListener("keydown", (e: KeyboardEvent) => {
  if (e.code == "Escape") {
    cancel();
  }
});

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

// reference to <DatabaseLabelForm /> to call validate()
const labelForm = ref<InstanceType<typeof DatabaseLabelForm> | null>(null);

const isDbNameTemplateMode = computed((): boolean => {
  if (parseInt(project.value.uid, 10) === UNKNOWN_ID) return false;

  if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
    return false;
  }

  // true if dbNameTemplate is not empty
  return !!project.value.dbNameTemplate;
});

const allowCreate = computed(() => {
  // If we are not in template mode, none of labels are required
  // So we just treat this case as 'yes, valid'
  const isLabelValid = isDbNameTemplateMode.value
    ? labelForm.value?.validate()
    : true;
  return (
    !isEmpty(state.databaseName) &&
    validDatabaseOwnerName.value &&
    !isReservedName.value &&
    isLabelValid &&
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

// If environment has been specified, then we disallow changing it.
const allowEditEnvironment = computed(() => {
  return !props.environmentId;
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

useDBNameTemplateInputState(project, {
  databaseName: toRef(state, "databaseName"),
  labels: toRef(state, "labels"),
});

const selectProject = (projectId: string) => {
  state.projectId = projectId;
};

const selectEnvironment = (environmentId: string) => {
  state.environmentId = environmentId;
};

const selectInstance = (instanceId: string | undefined) => {
  if (!instanceId) return;
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

const create = async () => {
  if (!allowCreate.value) {
    return;
  }

  let newIssue: IssueCreate;

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
  // Do not submit non-selected optional labels
  const labels = Object.keys(state.labels)
    .map((key) => {
      const value = state.labels[key];
      return { key, value };
    })
    .filter((kv) => !!kv.value);

  const createDatabaseContext: CreateDatabaseContext = {
    instanceId,
    databaseName: databaseName,
    tableName: tableName,
    owner,
    characterSet:
      state.characterSet ||
      defaultCharsetOfEngineV1(selectedInstance.value.engine),
    collation:
      state.collation ||
      defaultCollationOfEngineV1(selectedInstance.value.engine),
    cluster: state.cluster,
    labels: JSON.stringify(labels),
  };

  if (props.backup) {
    // If props.backup is specified, we create a PITR issue
    // with createDatabaseContext
    const { instance, database } = extractDatabaseResourceName(
      props.backup.name
    );
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      `instances/${instance}/databases/${database}`
    );
    const createContext: PITRContext = {
      databaseId: Number(db.uid),
      backupId: Number(props.backup.uid),
      createDatabaseContext,
    };
    newIssue = {
      name: `Create database '${databaseName}' from backup '${props.backup.name}'`,
      type: "bb.issue.database.restore.pitr",
      description: `Creating database '${databaseName}' from backup '${props.backup.name}'`,
      assigneeId: parseInt(state.assigneeId!, 10),
      projectId: parseInt(state.projectId!, 10),
      pipeline: {
        stageList: [],
        name: "",
      },
      createContext,
      payload: {},
    };
  } else {
    // Otherwise we create a simple database.create issue.
    newIssue = {
      name: `Create database '${databaseName}'`,
      type: "bb.issue.database.create",
      description: "",
      assigneeId: parseInt(state.assigneeId!, 10),
      projectId: parseInt(state.projectId!, 10),
      pipeline: {
        stageList: [],
        name: "",
      },
      createContext: createDatabaseContext,
      payload: {},
    };
  }

  state.creating = true;
  useIssueStore()
    .createIssue(newIssue)
    .then(
      (createdIssue) => {
        router.push(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
      },
      () => {
        state.creating = false;
      }
    );
};

// update `state.labelList` when selected Environment changed
watchEffect(() => {
  const envId = state.environmentId;
  const { labels } = state;
  const key = "bb.environment";
  if (envId) {
    const env = useEnvironmentV1Store().getEnvironmentByUID(envId);
    const resourceId = extractEnvironmentResourceName(env.name);
    labels[key] = resourceId;
  } else {
    delete labels[key];
  }
});

defineExpose({
  allowCreate,
  cancel,
  create,
});
</script>
