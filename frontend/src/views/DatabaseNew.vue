<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>

<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
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
          class="mt-1"
          id="environment"
          name="environment"
          :disabled="!allowEditEnvironment"
          :selectedId="state.environmentId"
          @select-environment-id="selectEnvironment"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <div class="flex flex-row items-center">
          <label for="instance" class="textlabel">
            Instance <span class="text-red-600">*</span>
          </label>
          <router-link
            :to="
              state.environmentId
                ? `/instance?environment=${state.environmentId}`
                : '/instance'
            "
            class="ml-2 text-sm normal-link"
          >
            List
          </router-link>
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
          <template v-if="instanceLink">
            <router-link :to="instanceLink" class="ml-2 normal-link text-sm">
              View
            </router-link>
          </template>
        </div>
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="database" class="textlabel">
          New database name <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-1 w-full"
          :disabled="!allowEditDatabaseName"
          @input="changeDatabaseName"
          v-model="state.databaseName"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="task" class="textlabel"> Task </label>
        <div class="mt-1 relative rounded-md shadow-sm">
          <router-link
            v-if="$router.currentRoute.value.query.task"
            :to="`/task/${$router.currentRoute.value.query.task}`"
            class="normal-link"
          >
            {{ `task/${$router.currentRoute.value.query.task}` }}
          </router-link>
          <template v-else>
            <div
              class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
            >
              <span class="text-accent font-semibold sm:text-sm">task/</span>
            </div>
            <div class="flex flex-row space-x-2 items-center">
              <input
                class="textfield w-full pl-12"
                id="task"
                name="task"
                type="number"
                placeholder="Your task id (e.g. 1234)"
                :disabled="!allowEditTask"
                v-model="state.taskId"
              />
              <template v-if="taskLink">
                <router-link
                  :to="taskLink"
                  target="_blank"
                  class="ml-2 normal-link text-sm"
                >
                  View
                </router-link>
              </template>
            </div>
          </template>
        </div>
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
  </form>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import InstanceSelect from "../components/InstanceSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import ProjectSelect from "../components/ProjectSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { instanceSlug, databaseSlug, taskSlug } from "../utils";
import {
  Task,
  TaskId,
  TaskType,
  DataSourceNew,
  EnvironmentId,
  InstanceId,
  PrincipalId,
  ProjectId,
} from "../types";
import { TaskField, templateForType } from "../plugins";
import { isEqual } from "lodash";

interface LocalState {
  projectId?: ProjectId;
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  databaseName?: string;
  taskId?: TaskId;
  fromTaskType?: TaskType;
}

export default {
  name: "DatabaseNew",
  props: {},
  components: {
    InstanceSelect,
    EnvironmentSelect,
    ProjectSelect,
    PrincipalSelect,
  },
  setup(props, ctx) {
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

    const state = reactive<LocalState>({
      projectId: router.currentRoute.value.query.project
        ? (router.currentRoute.value.query.project as ProjectId)
        : undefined,
      environmentId: router.currentRoute.value.query.environment
        ? (router.currentRoute.value.query.environment as EnvironmentId)
        : undefined,
      instanceId: router.currentRoute.value.query.instance
        ? (router.currentRoute.value.query.instance as InstanceId)
        : undefined,
      databaseName: router.currentRoute.value.query.name as string,
      taskId: router.currentRoute.value.query.task as TaskId,
      fromTaskType: router.currentRoute.value.query.from as TaskType,
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.databaseName) &&
        state.projectId &&
        state.environmentId &&
        state.instanceId
      );
    });

    // If it's from database request task, we disallow changing the preset field.
    // This is to prevent accidentally changing the requested field.
    const allowEditProject = computed(() => {
      return (
        state.fromTaskType != "bytebase.database.create" || !state.projectId
      );
    });

    const allowEditEnvironment = computed(() => {
      return (
        state.fromTaskType != "bytebase.database.create" || !state.environmentId
      );
    });

    const allowEditDatabaseName = computed(() => {
      return (
        state.fromTaskType != "bytebase.database.create" || !state.databaseName
      );
    });

    const allowEditTask = computed(() => {
      return state.fromTaskType != "bytebase.database.create" || !state.taskId;
    });

    const instanceLink = computed((): string => {
      if (state.instanceId) {
        const instance = store.getters["instance/instanceById"](
          state.instanceId
        );
        return `/instance/${instanceSlug(instance)}`;
      }
      return "";
    });

    const taskLink = computed((): string => {
      if (state.taskId) {
        // We intentionally not to validate whether the taskId is legit, we will do the validation
        // when actually trying to create the database.
        return `/task/${state.taskId}`;
      }
      return "";
    });

    const selectProject = (projectId: ProjectId) => {
      state.projectId = projectId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (projectId) {
        query.projectId = projectId;
      } else {
        delete query["project"];
      }
      router.replace({
        name: "workspace.database.create",
        query: {
          ...router.currentRoute.value.query,
          project: projectId,
        },
      });
    };

    const selectEnvironment = (environmentId: EnvironmentId) => {
      state.environmentId = environmentId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (environmentId) {
        query.environmentId = environmentId;
      } else {
        delete query["instance"];
      }
      router.replace({
        name: "workspace.database.create",
        query: {
          ...router.currentRoute.value.query,
          environment: environmentId,
        },
      });
    };

    const selectInstance = (instanceId: InstanceId) => {
      state.instanceId = instanceId;
      const query = cloneDeep(router.currentRoute.value.query);
      if (instanceId) {
        query.instance = instanceId;
      } else {
        delete query["instance"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const changeDatabaseName = () => {
      const query = cloneDeep(router.currentRoute.value.query);
      if (!isEmpty(state.databaseName)) {
        query.name = state.databaseName!;
      } else {
        delete query["name"];
      }
      router.replace({
        name: "workspace.database.create",
        query,
      });
    };

    const cancel = () => {
      router.go(-1);
    };

    const create = async () => {
      // If taskId id provided, we check its existence first.
      // We only set the taskId if it's valid.
      let linkedTask: Task | undefined = undefined;
      if (state.taskId) {
        try {
          linkedTask = await store.dispatch("task/fetchTaskById", state.taskId);
        } catch (err) {
          console.warn(`Unable to fetch linked task id ${state.taskId}`, err);
        }
      }

      // Create database
      const createdDatabase = await store.dispatch("database/createDatabase", {
        name: state.databaseName,
        projectId: state.projectId,
        instanceId: state.instanceId,
        creatorId: currentUser.value.id,
        taskId: linkedTask?.id,
      });

      // Redirect to the created database.
      router.push(`/db/${databaseSlug(createdDatabase)}`);

      // If a valid task id is provided, we will set the database output field
      // if it's not set before. This is based on the assumption that user creates
      // the database to fullfill that particular task (e.g. completing the request db workflow)
      // TODO: If there is no applicable database field, we should also link the database to the task.
      if (linkedTask) {
        const template = templateForType(linkedTask.type);
        if (template) {
          const databaseOutputField = template.fieldList.find(
            (item: TaskField) =>
              item.category == "OUTPUT" && item.type == "Database"
          );

          if (databaseOutputField) {
            const payload = cloneDeep(linkedTask.payload);
            // Only sets the value if it's empty to prevent accidentally overwriting
            // the existing legit data (e.g. someone provides a wrong task id)
            if (
              databaseOutputField &&
              isEmpty(payload[databaseOutputField.id])
            ) {
              payload[databaseOutputField.id] = createdDatabase.id;
            }

            if (!isEqual(payload, linkedTask.payload)) {
              await store.dispatch("task/patchTask", {
                taskId: linkedTask.id,
                taskPatch: {
                  payload,
                  updaterId: currentUser.value.id,
                },
              });
            }
          }
        }
      }

      store.dispatch("uistate/saveIntroStateByKey", {
        key: "database.create",
        newState: true,
      });

      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "SUCCESS",
        title: `Succesfully created database '${createdDatabase.name}'.`,
        description: linkedTask
          ? `We also linked the created database to the requested task '${linkedTask.name}'.`
          : "",
        link: linkedTask
          ? `/task/${taskSlug(linkedTask.name, linkedTask.id)}`
          : undefined,
        linkTitle: linkedTask ? "View task" : undefined,
        manualHide: linkedTask != undefined,
      });
    };

    return {
      state,
      allowCreate,
      allowEditProject,
      allowEditEnvironment,
      allowEditDatabaseName,
      allowEditTask,
      instanceLink,
      taskLink,
      selectProject,
      selectEnvironment,
      selectInstance,
      changeDatabaseName,
      cancel,
      create,
    };
  },
};
</script>
