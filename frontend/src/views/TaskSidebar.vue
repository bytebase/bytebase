<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="grid gap-y-6 gap-x-6 grid-cols-3">
      <template v-if="!$props.new">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Status
        </h2>
        <div class="col-span-2">
          <span class="flex items-center space-x-2">
            <TaskStatusIcon :taskStatus="task.status" :size="'normal'" />
            <span class="text-main capitalize">
              {{ task.status.toLowerCase() }}
            </span>
          </span>
        </div>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Assignee<span v-if="$props.new" class="text-red-600">*</span>
      </h2>
      <!-- Only DBA can be assigned to the task -->
      <div class="col-span-2">
        <PrincipalSelect
          :disabled="!allowEditAssignee"
          :selectedId="$props.new ? task.assigneeId : task.assignee?.id"
          :allowedRoleList="['DBA']"
          @select-principal-id="
            (principalId) => {
              $emit('update-assignee-id', principalId);
            }
          "
        />
      </div>

      <template v-for="(field, index) in fieldList" :key="index">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ field.name }}
          <span v-if="field.required" class="text-red-600">*</span>
        </h2>
        <div class="col-span-2">
          <template v-if="field.type == 'String'">
            <BBTextField
              class="mt-4 text-sm"
              :disabled="!allowEdit"
              :required="true"
              :value="fieldValue(field)"
              :placeholder="field.placeholder"
              @end-editing="(text) => trySaveCustomField(field, text)"
            />
          </template>
          <template v-else-if="field.type == 'Environment'">
            <EnvironmentSelect
              :disabled="!allowEdit"
              :name="field.id"
              :selectedId="fieldValue(field)"
              :selectDefault="false"
              @select-environment-id="
                (environmentId) => {
                  trySaveCustomField(field, environmentId);
                }
              "
            />
          </template>
          <template v-else-if="field.type == 'Project'">
            <ProjectSelect
              :disabled="!allowEdit"
              :name="field.id"
              :selectedId="fieldValue(field)"
              :selectDefault="false"
              @select-project-id="
                (projectId) => {
                  trySaveCustomField(field, projectId);
                }
              "
            />
          </template>
          <template v-else-if="field.type == 'Database'">
            <DatabaseSelect
              class="w-full"
              :disabled="!allowEdit"
              :selectedId="fieldValue(field)"
              :mode="'USER'"
              @select-database-id="
                (databaseId) => {
                  trySaveDatabaseId(field, databaseId);
                }
              "
            />
          </template>
          <template v-else-if="field.type == 'NewDatabase'">
            <BBTextField
              type="text"
              class="w-full text-sm"
              :disabled="!allowEdit"
              :required="true"
              :value="fieldValue(field)"
              :placeholder="field.placeholder"
              @end-editing="(text) => trySaveCustomField(field, text)"
            />
          </template>
          <template v-else-if="field.type == 'Boolean'">
            <BBSwitch
              :disabled="!allowEdit"
              :value="fieldValue(field)"
              @toggle="
                (on) => {
                  trySaveCustomField(field, on);
                }
              "
            />
          </template>
        </div>
      </template>
    </div>
    <div
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-6 grid-cols-3"
    >
      <template v-if="database.id != UNKNOWN_ID">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Database<span v-if="$props.new" class="text-red-600">*</span>
        </h2>
        <router-link
          :to="`/db/${databaseSlug(database)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          {{ database.name }}
        </router-link>

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Environment
        </h2>
        <router-link
          :to="`/environment/${environmentSlug(environment)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          {{ environmentName(environment) }}
        </router-link>

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Project
        </h2>
        <router-link
          :to="`/project/${projectSlug(project)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          {{ projectName(project) }}
        </router-link>
      </template>

      <template v-if="!$props.new">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Updated
        </h2>
        <span class="textfield col-span-2">
          {{ moment(task.updatedTs).format("LLL") }}</span
        >

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Created
        </h2>
        <span class="textfield col-span-2">
          {{ moment(task.createdTs).format("LLL") }}</span
        >
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Creator
        </h2>
        <ul class="col-span-2">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar :size="'small'" :username="task.creator.name" />
            </div>
            <router-link
              :to="`/u/${task.creator.id}`"
              class="text-sm font-medium text-main hover:underline"
            >
              {{ task.creator.name }}
            </router-link>
          </li>
        </ul>
      </template>
    </div>
    <div
      v-if="!$props.new"
      class="mt-6 border-t border-block-border pt-6 grid gap-y-4 gap-x-6 grid-cols-3"
    >
      <h2
        class="textlabel flex items-center col-span-1 col-start-1 whitespace-nowrap"
      >
        {{
          task.subscriberList.length +
          (task.subscriberList.length > 1 ? " subscribers" : " subscriber")
        }}
      </h2>
      <div v-if="subscriberList.length > 0" class="col-span-3 col-start-1">
        <div class="flex space-x-1">
          <template v-for="(subscriber, index) in subscriberList" :key="index">
            <router-link :to="`/u/${subscriber.id}`" class="hover:opacity-75">
              <BBAvatar :size="'small'" :username="subscriber.name" />
            </router-link>
          </template>
        </div>
      </div>
      <button
        type="button"
        class="btn-normal items-center col-span-3 col-start-1"
        @click.prevent="toggleSubscription"
      >
        <span class="w-full">
          <svg
            class="h-5 w-5 text-control inline -mt-0.5 mr-1"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              v-if="isCurrentUserSubscribed"
              fill-rule="evenodd"
              d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z"
              clip-rule="evenodd"
            ></path>
            <path
              v-else
              d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z"
            ></path></svg
          >{{ isCurrentUserSubscribed ? "Unsubscribe" : "Subscribe" }}</span
        >
      </button>
    </div>
  </aside>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import ProjectSelect from "../components/ProjectSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { TaskField, DatabaseFieldPayload } from "../plugins";
import {
  Database,
  DatabaseId,
  Environment,
  EnvironmentId,
  Principal,
  Project,
  Task,
  TaskNew,
  UNKNOWN_ID,
} from "../types";
import { activeDatabaseId, isDBAOrOwner } from "../utils";

interface LocalState {}

export default {
  name: "TaskSidebar",
  emits: [
    "update-assignee-id",
    "update-subscriber-list",
    "update-custom-field",
  ],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task | TaskNew>,
    },
    new: {
      required: true,
      type: Boolean,
    },
    fieldList: {
      required: true,
      type: Object as PropType<TaskField[]>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  components: {
    DatabaseSelect,
    ProjectSelect,
    EnvironmentSelect,
    PrincipalSelect,
    TaskStatusIcon,
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const fieldValue = (field: TaskField): string | DatabaseFieldPayload => {
      return props.task.payload[field.id];
    };

    const database = computed(
      (): Database => {
        const databaseId = props.new
          ? (props.task as TaskNew).stageList[0].databaseId
          : activeDatabaseId(props.task as Task);
        return store.getters["database/databaseById"](databaseId);
      }
    );

    const project = computed(
      (): Project => {
        // For new task, we derive the project from the 1st stage's database.
        // For existing task, we use task's project. We can't derive from the stage's database because
        // database may be transferred to a different project after creating the task.
        return props.new
          ? database.value.project
          : (props.task as Task).project;
      }
    );

    const environment = computed(
      (): Environment => {
        return database.value.instance.environment;
      }
    );

    const isCurrentUserSubscribed = computed((): boolean => {
      for (const principal of (props.task as Task).subscriberList) {
        if (currentUser.value.id == principal.id) {
          return true;
        }
      }
      return false;
    });

    const subscriberList = computed((): Principal[] => {
      const list: Principal[] = [];
      (props.task as Task).subscriberList.forEach((principal: Principal) => {
        // Put the current user at the front if in the list.
        if (currentUser.value.id == principal.id) {
          list.unshift(principal);
        } else {
          list.push(principal);
        }
      });
      return list;
    });

    const allowEditAssignee = computed(() => {
      // We allow the current assignee or DBA to re-assign the task.
      // Though only DBA can be assigned to the task, the current
      // assignee might not have DBA role in case its role is revoked after
      // being assigned to the task.
      return (
        props.new ||
        ((props.task as Task).status == "OPEN" &&
          (currentUser.value.id == (props.task as Task).assignee?.id ||
            isDBAOrOwner(currentUser.value.role)))
      );
    });

    const trySaveCustomField = (
      field: TaskField,
      value: string | EnvironmentId | DatabaseFieldPayload
    ) => {
      if (!isEqual(value, fieldValue(field))) {
        emit("update-custom-field", field, value);
      }
    };

    const trySaveDatabaseNew = (field: TaskField, isNew: boolean) => {
      // Do a deep clone to prevent caller accidentally changes the original data.
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.isNew = isNew;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseName = (field: TaskField, value: string) => {
      // Do a deep clone to prevent caller accidentally changes the original data.
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.name = value;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseId = (field: TaskField, value: DatabaseId) => {
      // Do a deep clone to prevent caller accidentally changes the original data.
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.id = value;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseReadOnly = (field: TaskField, value: boolean) => {
      // Do a deep clone to prevent caller accidentally changes the original data.
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.readOnly = value;
      trySaveCustomField(field, payload);
    };

    const toggleSubscription = () => {
      const list = cloneDeep((props.task as Task).subscriberList);
      if (isCurrentUserSubscribed.value) {
        const index = (props.task as Task).subscriberList.findIndex(
          (item: Principal) => {
            return item.id == currentUser.value.id;
          }
        );
        if (index >= 0) {
          list.splice(index, 1);
        }
      } else {
        list.push(currentUser.value);
      }
      emit(
        "update-subscriber-list",
        list.map((item: Principal) => {
          return item.id;
        })
      );
    };

    return {
      UNKNOWN_ID,
      state,
      allowEditAssignee,
      fieldValue,
      environment,
      database,
      project,
      isCurrentUserSubscribed,
      subscriberList,
      trySaveCustomField,
      trySaveDatabaseNew,
      trySaveDatabaseName,
      trySaveDatabaseId,
      trySaveDatabaseReadOnly,
      toggleSubscription,
    };
  },
};
</script>
