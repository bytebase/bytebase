<template>
  <div class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          New database name <span class="text-red-600">*</span>
        </label>
        <input
          id="name"
          v-model="state.databaseName"
          required
          name="name"
          type="text"
          class="textfield mt-1 w-full"
        />
        <span v-if="isReservedName" class="text-red-600"
          >{{ state.databaseName }} is a reserved name</span
        >
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="project" class="textlabel">
          Project <span style="color: red">*</span>
        </label>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <ProjectSelect
          id="project"
          class="mt-1"
          name="project"
          :disabled="!allowEditProject"
          :selectedId="state.projectId"
          @select-project-id="selectProject"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <EnvironmentSelect
          id="environment"
          class="mt-1 w-full"
          name="environment"
          :disabled="!allowEditEnvironment"
          :selectedId="state.environmentId"
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
            Instance <span class="text-red-600">*</span>
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
                ? "Encoding"
                : "Character set"
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
          <label for="collation" class="textlabel"> Collation </label>
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
          Assignee <span class="text-red-600">*</span>
        </label>
        <!-- DBA and Owner always have all access, so we only need to grant to developer -->
        <!-- eslint-disable vue/attribute-hyphenation -->
        <MemberSelect
          id="user"
          class="mt-1"
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
        Cancel
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="create"
      >
        Create
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import {
  computed,
  reactive,
  onMounted,
  onUnmounted,
  PropType,
  watchEffect,
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
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
  StageCreate,
  defaultCharset,
  defaultCollation,
  unknown,
} from "../types";
import { isDBAOrOwner, issueSlug } from "../utils";

interface LocalState {
  projectId?: ProjectId;
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  databaseName?: string;
  characterSet: string;
  collation: string;
  assigneeId?: PrincipalId;
}

export default {
  name: "CreateDatabasePrepForm",
  components: {
    InstanceSelect,
    EnvironmentSelect,
    ProjectSelect,
    MemberSelect,
    InstanceEngineIcon,
  },
  props: {
    projectId: {
      type: Number as PropType<ProjectId>,
    },
    environmentId: {
      type: Number as PropType<EnvironmentId>,
    },
    instanceId: {
      type: Number as PropType<InstanceId>,
    },
    // If specified, then we are creating a database from the backup.
    backup: {
      type: Object as PropType<Backup>,
    },
  },
  emits: ["dismiss"],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    // Refresh the instance list
    const prepareInstanceList = () => {
      store.dispatch("instance/fetchInstanceList");
    };

    watchEffect(prepareInstanceList);

    const showAssigneeSelect = computed(() => {
      return !isDBAOrOwner(currentUser.value.role);
    });

    const state = reactive<LocalState>({
      projectId: props.projectId,
      environmentId: props.environmentId,
      instanceId: props.instanceId,
      characterSet: "",
      collation: "",
      assigneeId: showAssigneeSelect.value ? undefined : SYSTEM_BOT_ID,
    });

    const isReservedName = computed(() => {
      return state.databaseName?.toLowerCase() == "bytebase";
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.databaseName) &&
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

    // If environment has been specified, then we disallow changing it.
    const allowEditEnvironment = computed(() => {
      return !props.environmentId;
    });

    // If instance has been specified, then we disallow changing it.
    const allowEditInstance = computed(() => {
      return !props.instanceId;
    });

    const selectedInstance = computed(() => {
      return state.instanceId
        ? store.getters["instance/instanceById"](state.instanceId)
        : unknown("INSTANCE");
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
      const stageList: StageCreate[] = [
        {
          name: "Create database",
          environmentId: state.environmentId!,
          taskList: [
            {
              name: `Create database '${state.databaseName}'`,
              // If current user is DBA or Owner, then the created task will start automatically,
              // otherwise, it will require approval.
              status: isDBAOrOwner(currentUser.value.role)
                ? "PENDING"
                : "PENDING_APPROVAL",
              type: "bb.task.database.create",
              instanceId: state.instanceId!,
              // statement is derived by backend.
              statement: ``,
              rollbackStatement: "",
              databaseName: state.databaseName,
              characterSet:
                state.characterSet ||
                defaultCharset(selectedInstance.value.engine),
              collation:
                state.collation ||
                defaultCollation(selectedInstance.value.engine),
            },
          ],
        },
      ];

      // If backup is specified, then we add an additional stage to restore the backup to the newly created database.
      if (props.backup) {
        stageList.push({
          name: "Restore backup",
          environmentId: state.environmentId!,
          taskList: [
            {
              name: `Restore backup '${props.backup.name}'`,
              // Use "PENDING" here since we consider the required approval has already been granted in the first stage.
              status: "PENDING",
              type: "bb.task.database.restore",
              instanceId: state.instanceId!,
              statement: "",
              rollbackStatement: "",
              databaseName: state.databaseName,
              backupId: props.backup.id,
            },
          ],
        });
      }
      const newIssue: IssueCreate = props.backup
        ? {
            name: `Create database '${state.databaseName}' from backup '${props.backup.name}'`,
            type: "bb.issue.database.create",
            description: `Creating database from backup '${props.backup.name}'`,
            assigneeId: state.assigneeId!,
            projectId: state.projectId!,
            pipeline: {
              stageList,
              name: `Pipeline - Create database '${state.databaseName}' from backup '${props.backup.name}'`,
            },
            payload: {},
          }
        : {
            name: `Create database '${state.databaseName}'`,
            type: "bb.issue.database.create",
            description: "",
            assigneeId: state.assigneeId!,
            projectId: state.projectId!,
            pipeline: {
              stageList,
              name: `Pipeline - Create database ${state.databaseName}`,
            },
            payload: {},
          };
      store.dispatch("issue/createIssue", newIssue).then((createdIssue) => {
        router.push(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
      });
    };

    return {
      defaultCharset,
      defaultCollation,
      state,
      isReservedName,
      allowCreate,
      allowEditProject,
      allowEditEnvironment,
      allowEditInstance,
      selectedInstance,
      showAssigneeSelect,
      selectProject,
      selectEnvironment,
      selectInstance,
      selectAssignee,
      cancel,
      create,
    };
  },
};
</script>
