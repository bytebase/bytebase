<template>
  <div class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-2 col-start-2 w-64">
        <label for="name" class="textlabel">
          New database name <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-1 w-full"
          v-model="state.databaseName"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="project" class="textlabel">
          Project <span style="color: red">*</span>
        </label>
        <ProjectSelect
          class="mt-1"
          id="project"
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
        <EnvironmentSelect
          class="mt-1 w-full"
          id="environment"
          name="environment"
          :selectedId="state.environmentId"
          @select-environment-id="selectEnvironment"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <div class="flex flex-row items-center">
          <label for="instance" class="textlabel">
            Instance <span class="text-red-600">*</span>
          </label>
        </div>
        <div class="flex flex-row space-x-2 items-center">
          <InstanceSelect
            class="mt-1"
            id="instance"
            name="instance"
            :selectedId="state.instanceId"
            :environmentId="state.environmentId"
            @select-instance-id="selectInstance"
          />
        </div>
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="charset" class="textlabel">
          Character set <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="charset"
          name="charset"
          type="text"
          class="textfield mt-1 w-full"
          v-model="state.characterSet"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="collation" class="textlabel">
          Collation <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="collation"
          name="collation"
          type="text"
          class="textfield mt-1 w-full"
          v-model="state.collation"
        />
      </div>

      <div v-if="showAssigneeSelect" class="col-span-2 col-start-2 w-64">
        <label for="user" class="textlabel">
          Assignee <span class="text-red-600">*</span>
        </label>
        <!-- DBA and Owner always have all access, so we only need to grant to developer -->
        <MemberSelect
          class="mt-1"
          id="user"
          name="user"
          :allowedRoleList="['OWNER', 'DBA']"
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
import {
  EnvironmentId,
  InstanceId,
  ProjectId,
  IssueCreate,
  SYSTEM_BOT_ID,
  PrincipalId,
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
  emits: ["dismiss"],
  props: {
    projectId: {
      type: Number as PropType<ProjectId>,
    },
  },
  components: {
    InstanceSelect,
    EnvironmentSelect,
    ProjectSelect,
    MemberSelect,
  },
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
      characterSet: "utf8mb4",
      collation: "utf8mb4_0900_ai_ci",
      assigneeId: showAssigneeSelect.value ? undefined : SYSTEM_BOT_ID,
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.databaseName) &&
        state.projectId &&
        state.environmentId &&
        state.instanceId &&
        !isEmpty(state.characterSet) &&
        !isEmpty(state.collation) &&
        state.assigneeId
      );
    });

    // If project has been specified, then we disallow changing it.
    const allowEditProject = computed(() => {
      return !props.projectId;
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
      const newIssue: IssueCreate = {
        name: `Create database ${state.databaseName}`,
        type: "bb.issue.database.create",
        description: "",
        assigneeId: state.assigneeId,
        projectId: state.projectId!,
        pipeline: {
          stageList: [
            {
              name: "Create database",
              environmentId: state.environmentId!,
              taskList: [
                {
                  name: `Create database ${state.databaseName}`,
                  // If current user is DBA or Owner, then the created task will start automatically,
                  // otherwise, it will require approval.
                  status: isDBAOrOwner(currentUser.value.role)
                    ? "PENDING"
                    : "PENDING_APPROVAL",
                  type: "bb.task.database.create",
                  instanceId: state.instanceId!,
                  statement: `CREATE DATABASE \`${state.databaseName}\`\nCHARACTER SET ${state.characterSet} COLLATE ${state.collation}`,
                  rollbackStatement: "",
                  databaseName: state.databaseName,
                  characterSet: state.characterSet,
                  collation: state.collation,
                },
              ],
            },
          ],
          name: `Pipeline - Create database ${state.databaseName}`,
        },
        payload: {},
      };
      store.dispatch("issue/createIssue", newIssue).then((createdIssue) => {
        router.push(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
      });
    };

    return {
      state,
      allowCreate,
      allowEditProject,
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
