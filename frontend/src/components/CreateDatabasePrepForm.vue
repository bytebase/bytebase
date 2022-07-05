<template>
  <div class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-2 col-start-2 w-64">
        <label for="project" class="textlabel">
          {{ $t("common.project") }} <span style="color: red">*</span>
        </label>
        <ProjectSelect
          id="project"
          class="mt-1"
          name="project"
          :disabled="!allowEditProject"
          :selected-id="state.projectId"
          @select-project-id="selectProject"
        />
      </div>

      <!-- Providing a preview of generated database name in template mode -->
      <div v-if="isDbNameTemplateMode" class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          {{ $t("create-db.generated-database-name") }}
          <NTooltip trigger="hover" placement="top">
            <template #trigger>
              <heroicons-outline:question-mark-circle
                class="w-4 h-4 inline-block"
              />
            </template>
            <div class="whitespace-nowrap">
              {{
                $t("create-db.db-name-generated-by-template", {
                  template: project.dbNameTemplate,
                })
              }}
            </div>
          </NTooltip>
        </label>
        <input
          id="name"
          disabled
          name="name"
          type="text"
          class="textfield mt-1 w-full"
          :value="generatedDatabaseName"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          {{ $t("create-db.new-database-name") }}
          <span class="text-red-600">*</span>
        </label>
        <input
          id="name"
          v-model="state.databaseName"
          required
          name="name"
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
      </div>

      <div v-if="requireDatabaseOwnerName" class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          {{ $t("create-db.database-owner-name") }}
          <span class="text-red-600">*</span>
        </label>
        <input
          id="name"
          v-model="state.databaseOwnerName"
          required
          name="ownerName"
          type="text"
          class="textfield mt-1 w-full"
        />
      </div>

      <div
        v-if="selectedInstance.engine == 'CLICKHOUSE'"
        class="col-span-2 col-start-2 w-64"
      >
        <label for="name" class="textlabel">
          {{ $t("create-db.cluster") }}
          <span class="text-red-600">*</span>
        </label>
        <input
          id="name"
          v-model="state.cluster"
          required
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
        :label-list="state.labelList"
        filter="required"
      />

      <div class="col-span-2 col-start-2 w-64">
        <label for="environment" class="textlabel">
          {{ $t("common.environment") }} <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          id="environment"
          class="mt-1 w-full"
          name="environment"
          :disabled="!allowEditEnvironment"
          :selected-id="state.environmentId"
          @select-environment-id="selectEnvironment"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <div class="flex flex-row items-center space-x-1">
          <InstanceEngineIcon
            v-if="state.instanceId"
            :instance="selectedInstance"
          />
          <label for="instance" class="textlabel">
            {{ $t("common.instance") }} <span class="text-red-600">*</span>
          </label>
        </div>
        <div class="flex flex-row space-x-2 items-center">
          <!-- eslint-disable vue/attribute-hyphenation -->
          <InstanceSelect
            id="instance"
            class="mt-1"
            name="instance"
            :disabled="!allowEditInstance"
            :selectedId="state.instanceId"
            :environmentId="state.environmentId"
            @select-instance-id="selectInstance"
          />
        </div>
      </div>

      <!-- Providing other dropdowns for optional labels as if they are normal optional props of DB -->
      <DatabaseLabelForm
        :project="project"
        :label-list="state.labelList"
        filter="optional"
      />

      <template
        v-if="
          selectedInstance.engine != 'CLICKHOUSE' &&
          selectedInstance.engine != 'SNOWFLAKE'
        "
      >
        <div class="col-span-2 col-start-2 w-64">
          <label for="charset" class="textlabel">
            {{
              selectedInstance.engine == "POSTGRES"
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
            :placeholder="defaultCharset(selectedInstance.engine)"
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
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
              defaultCollation(selectedInstance.engine) || 'default'
            "
          />
        </div>
      </template>

      <div v-if="showAssigneeSelect" class="col-span-2 col-start-2 w-64">
        <label for="user" class="textlabel">
          {{ $t("common.assignee") }} <span class="text-red-600">*</span>
        </label>
        <!-- DBA and Owner always have all access, so we only need to grant to developer -->
        <!-- eslint-disable vue/attribute-hyphenation -->
        <MemberSelect
          id="user"
          class="mt-1 w-full"
          name="user"
          :allowed-role-list="['OWNER', 'DBA']"
          :selectedId="state.assigneeId"
          :placeholder="'Select assignee'"
          @select-principal-id="selectAssignee"
        />
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="create"
      >
        {{ $t("common.create") }}
      </button>
    </div>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts">
import {
  computed,
  reactive,
  PropType,
  watchEffect,
  defineComponent,
  ref,
} from "vue";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash-es";
import { NTooltip } from "naive-ui";
import { DatabaseLabelForm } from "./CreateDatabasePrepForm/";
import InstanceSelect from "../components/InstanceSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import ProjectSelect from "../components/ProjectSelect.vue";
import MemberSelect from "../components/MemberSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import {
  EnvironmentId,
  InstanceId,
  ProjectId,
  IssueCreate,
  SYSTEM_BOT_ID,
  PrincipalId,
  Backup,
  defaultCharset,
  defaultCollation,
  unknown,
  Project,
  DatabaseLabel,
  CreateDatabaseContext,
  UNKNOWN_ID,
  Instance,
} from "../types";
import {
  buildDatabaseNameByTemplateAndLabelList,
  isDBAOrOwner,
  issueSlug,
} from "../utils";
import { useEventListener } from "@vueuse/core";
import {
  hasFeature,
  useCurrentUser,
  useEnvironmentStore,
  useInstanceStore,
  useIssueStore,
  useProjectStore,
} from "@/store";

interface LocalState {
  projectId?: ProjectId;
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  labelList: DatabaseLabel[];
  databaseName: string;
  databaseOwnerName: string;
  characterSet: string;
  collation: string;
  cluster: string;
  assigneeId?: PrincipalId;
  showFeatureModal: boolean;
}

export default defineComponent({
  name: "CreateDatabasePrepForm",
  components: {
    NTooltip,
    InstanceSelect,
    EnvironmentSelect,
    ProjectSelect,
    MemberSelect,
    InstanceEngineIcon,
    DatabaseLabelForm,
  },
  props: {
    projectId: {
      type: Number as PropType<ProjectId>,
      default: undefined,
    },
    environmentId: {
      type: Number as PropType<EnvironmentId>,
      default: undefined,
    },
    instanceId: {
      type: Number as PropType<InstanceId>,
      default: undefined,
    },
    // If specified, then we are creating a database from the backup.
    backup: {
      type: Object as PropType<Backup>,
      default: undefined,
    },
  },
  emits: ["dismiss"],
  setup(props, { emit }) {
    const instanceStore = useInstanceStore();
    const router = useRouter();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    useEventListener("keydown", (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        cancel();
      }
    });

    // Refresh the instance list
    const prepareInstanceList = () => {
      instanceStore.fetchInstanceList();
    };

    watchEffect(prepareInstanceList);

    const showAssigneeSelect = computed(() => {
      return !isDBAOrOwner(currentUser.value.role);
    });

    const state = reactive<LocalState>({
      databaseName: "",
      databaseOwnerName: "",
      projectId: props.projectId,
      environmentId: props.environmentId,
      instanceId: props.instanceId,
      labelList: [],
      characterSet: "",
      collation: "",
      cluster: "",
      assigneeId: showAssigneeSelect.value ? undefined : SYSTEM_BOT_ID,
      showFeatureModal: false,
    });

    const project = computed((): Project => {
      if (!state.projectId) return unknown("PROJECT") as Project;
      return projectStore.getProjectById(state.projectId) as Project;
    });

    const isReservedName = computed(() => {
      return state.databaseName.toLowerCase() == "bytebase";
    });

    const isTenantProject = computed((): boolean => {
      if (project.value.id === UNKNOWN_ID) return false;

      return project.value.tenantMode === "TENANT";
    });

    // reference to <DatabaseLabelForm /> to call validate()
    const labelForm = ref<InstanceType<typeof DatabaseLabelForm> | null>(null);

    const isDbNameTemplateMode = computed((): boolean => {
      if (project.value.id === UNKNOWN_ID) return false;

      // true if dbNameTemplate is not empty
      return !!project.value.dbNameTemplate;
    });

    const generatedDatabaseName = computed((): string => {
      if (!isDbNameTemplateMode.value) {
        // don't modify anything if we are not in template mode
        return state.databaseName;
      }

      return buildDatabaseNameByTemplateAndLabelList(
        project.value.dbNameTemplate,
        state.databaseName,
        state.labelList,
        true // keepEmpty: true to keep non-selected values as original placeholders
      );
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

    const selectedInstance = computed((): Instance => {
      return state.instanceId
        ? instanceStore.getInstanceById(state.instanceId)
        : (unknown("INSTANCE") as Instance);
    });

    const requireDatabaseOwnerName = computed((): boolean => {
      const instance = selectedInstance.value;
      if (instance.id === UNKNOWN_ID) {
        return false;
      }
      return instance.engine === "POSTGRES";
    });

    const validDatabaseOwnerName = computed((): boolean => {
      if (!requireDatabaseOwnerName.value) {
        return true;
      }

      return !isEmpty(state.databaseOwnerName);
    });

    const selectProject = (projectId: ProjectId) => {
      state.projectId = projectId;
    };

    const selectEnvironment = (environmentId: EnvironmentId) => {
      state.environmentId = environmentId;
    };

    const selectInstance = (instanceId: InstanceId) => {
      state.instanceId = instanceId;
    };

    const selectAssignee = (assigneeId: PrincipalId) => {
      state.assigneeId = assigneeId;
    };

    const cancel = () => {
      emit("dismiss");
    };

    const create = async () => {
      var newIssue: IssueCreate;

      const databaseName = isDbNameTemplateMode.value
        ? generatedDatabaseName.value
        : state.databaseName;
      const owner = requireDatabaseOwnerName.value
        ? state.databaseOwnerName
        : "";

      if (props.backup) {
        newIssue = {
          name: `Create database '${databaseName}' from backup '${props.backup.name}'`,
          type: "bb.issue.database.create",
          description: `Creating database '${databaseName}' from backup '${props.backup.name}'`,
          assigneeId: state.assigneeId!,
          projectId: state.projectId!,
          pipeline: {
            stageList: [],
            name: "",
          },
          createContext: {
            instanceId: state.instanceId!,
            databaseName: databaseName,
            owner,
            characterSet:
              state.characterSet ||
              defaultCharset(selectedInstance.value.engine),
            collation:
              state.collation ||
              defaultCollation(selectedInstance.value.engine),
            cluster: state.cluster,
            backupId: props.backup.id,
            backupName: props.backup.name,
          },
          payload: {},
        };
      } else {
        newIssue = {
          name: `Create database '${databaseName}'`,
          type: "bb.issue.database.create",
          description: "",
          assigneeId: state.assigneeId!,
          projectId: state.projectId!,
          pipeline: {
            stageList: [],
            name: "",
          },
          createContext: {
            instanceId: state.instanceId!,
            databaseName: databaseName,
            owner,
            characterSet:
              state.characterSet ||
              defaultCharset(selectedInstance.value.engine),
            collation:
              state.collation ||
              defaultCollation(selectedInstance.value.engine),
            cluster: state.cluster,
          },
          payload: {},
        };
      }
      if (isTenantProject.value) {
        if (!hasFeature("bb.feature.multi-tenancy")) {
          state.showFeatureModal = true;
          return;
        }
      }
      const context = newIssue.createContext as CreateDatabaseContext;
      // Do not submit non-selected optional labels
      const labelList = state.labelList.filter((label) => !!label.value);
      context.labels = JSON.stringify(labelList);
      useIssueStore()
        .createIssue(newIssue)
        .then((createdIssue) => {
          router.push(
            `/issue/${issueSlug(createdIssue.name, createdIssue.id)}`
          );
        });
    };

    // update `state.labelList` when selected Environment changed
    watchEffect(() => {
      const envId = state.environmentId;
      const { labelList } = state;
      const key = "bb.environment";
      const index = labelList.findIndex((label) => label.key === key);
      if (envId) {
        const env = useEnvironmentStore().getEnvironmentById(envId);
        if (index >= 0) labelList[index].value = env.name;
        else labelList.unshift({ key, value: env.name });
      } else {
        if (index >= 0) labelList.splice(index, 1);
      }
    });

    return {
      defaultCharset,
      defaultCollation,
      state,
      isReservedName,
      project,
      isTenantProject,
      isDbNameTemplateMode,
      generatedDatabaseName,
      labelForm,
      allowCreate,
      allowEditProject,
      allowEditEnvironment,
      allowEditInstance,
      selectedInstance,
      requireDatabaseOwnerName,
      showAssigneeSelect,
      selectProject,
      selectEnvironment,
      selectInstance,
      selectAssignee,
      cancel,
      create,
    };
  },
});
</script>
