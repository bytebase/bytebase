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
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          class="mt-1"
          id="environment"
          name="environment"
          :selectedId="state.environmentId"
          @select-environment-id="
            (environmentId) => {
              state.environmentId = environmentId;
            }
          "
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="instance" class="textlabel">
          Instance <span class="text-red-600">*</span>
        </label>
        <InstanceSelect
          class="mt-1"
          id="instance"
          name="instance"
          :selectedId="state.instanceId"
          :environmentId="state.environmentId"
          @select-instance-id="
            (instanceId) => {
              state.instanceId = instanceId;
            }
          "
        />
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
          v-model="state.databaseName"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="user" class="textlabel">
          Owner <span class="text-red-600">*</span>
        </label>
        <PrincipalSelect
          class="mt-1"
          id="owner"
          name="owner"
          :selectedId="state.ownerId"
          @select-principal-id="
            (principalId) => {
              state.ownerId = principalId;
            }
          "
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <label for="username" class="textlabel block"> Username </label>
        <!-- For mysql, username can be empty indicating anonymous user. 
            But it's a very bad practice to use anonymous user for admin operation,
            thus we make it REQUIRED here. -->
        <input
          id="username"
          name="username"
          type="text"
          class="textfield mt-1 w-full"
          v-model="state.username"
        />
      </div>

      <div class="col-span-2 col-start-2 w-64">
        <div class="flex flex-row items-center">
          <label for="password" class="textlabel block"> Password </label>
          <div class="ml-1 flex items-center">
            <button
              class="btn-icon"
              @click.prevent="state.showPassword = !state.showPassword"
            >
              <svg
                v-if="state.showPassword"
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                ></path>
              </svg>
              <svg
                v-else
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                ></path>
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                ></path>
              </svg>
            </button>
          </div>
        </div>
        <input
          id="password"
          name="password"
          autocomplete="off"
          :type="state.showPassword ? 'text' : 'password'"
          class="textfield mt-1 w-full"
          v-model="state.password"
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
            <input
              class="textfield w-full pl-12"
              id="task"
              name="task"
              type="number"
              placeholder="Your task id (e.g. 1234)"
              v-model="state.taskId"
            />
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
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { instanceSlug, databaseSlug, taskSlug } from "../utils";
import {
  Task,
  TaskId,
  Database,
  DataSourceNew,
  EnvironmentId,
  Environment,
  InstanceId,
  PrincipalId,
} from "../types";
import { TaskField, templateForType } from "../plugins";
import { isEqual } from "lodash";

interface LocalState {
  databaseName?: string;
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  ownerId: PrincipalId;
  taskId?: TaskId;
  username: string;
  passsword: string;
  showPassword: boolean;
}

export default {
  name: "DatabaseNew",
  emits: ["create", "cancel"],
  props: {},
  components: { InstanceSelect, EnvironmentSelect, PrincipalSelect },
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
      databaseName: router.currentRoute.value.query.name as string,
      environmentId: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentById"](
            router.currentRoute.value.query.environment
          )?.id
        : undefined,
      instanceId: router.currentRoute.value.query.instance
        ? store.getters["instance/instanceById"](
            router.currentRoute.value.query.instance
          )?.id
        : undefined,
      ownerId:
        (router.currentRoute.value.query.owner as PrincipalId) ||
        currentUser.value.id,
      taskId: router.currentRoute.value.query.task as TaskId,
      username: "",
      passsword: "",
      showPassword: false,
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.databaseName) &&
        state.environmentId &&
        state.instanceId &&
        state.ownerId
      );
    });

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
        instanceId: state.instanceId,
        ownerId: state.ownerId,
        creatorId: currentUser.value.id,
        taskId: linkedTask?.id,
      });

      // Create the default RW data source
      const newDataSource: DataSourceNew = {
        name: "Default RW",
        type: "RW",
        databaseId: createdDatabase.id,
        username: isEmpty(state.username) ? undefined : state.username,
        password: isEmpty(state.passsword) ? undefined : state.passsword,
        memberList: [
          {
            principalId: state.ownerId,
            taskId: linkedTask?.id,
          },
        ],
      };
      const createdDataSource = await store.dispatch(
        "dataSource/createDataSource",
        {
          instanceId: state.instanceId,
          newDataSource,
        }
      );

      router.push(
        `/instance/${instanceSlug(createdDatabase.instance)}/db/${databaseSlug(
          createdDatabase
        )}`
      );

      // If a valid task id is provided, we will set the database and data source output field
      // if it's not set before. This is based on the assumption that user creates
      // the database/data source to fullfill that particular task (e.g. completing
      // the request db workflow)
      if (linkedTask) {
        const template = templateForType(linkedTask.type);
        if (template) {
          const databaseOutputField = template.fieldList.find(
            (item: TaskField) =>
              item.category == "OUTPUT" && item.type == "Database"
          );

          const dataSourceOutputField = template.fieldList.find(
            (item: TaskField) =>
              item.category == "OUTPUT" && item.type == "DataSource"
          );

          if (databaseOutputField || dataSourceOutputField) {
            const payload = cloneDeep(linkedTask.payload);
            // Only sets the value if it's empty to prevent accidentally overwriting
            // the existing legit data (e.g. someone provides a wrong task id)
            if (
              databaseOutputField &&
              isEmpty(payload[databaseOutputField.id])
            ) {
              payload[databaseOutputField.id] = createdDatabase.id;
            }
            if (
              dataSourceOutputField &&
              isEmpty(payload[dataSourceOutputField.id])
            ) {
              payload[dataSourceOutputField.id] = createdDataSource.id;
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

      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "SUCCESS",
        title: `Succesfully created database '${createdDatabase.name}'.`,
        description: linkedTask
          ? `Task '${linkedTask.name}' is also linked with the created database and its data source.`
          : "Also created a default Read & Write data source.",
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
      cancel,
      create,
    };
  },
};
</script>
