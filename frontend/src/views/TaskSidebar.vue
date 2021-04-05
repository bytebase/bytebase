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
      <div class="col-span-2">
        <PrincipalSelect
          :disabled="!allowEditAssignee"
          :selectedId="$props.new ? task.assigneeId : task.assignee?.id"
          :allowAllRoles="false"
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
          <template v-else-if="field.type == 'Database'">
            <DatabaseSelect
              class="w-full"
              :disabled="!allowEdit"
              :selectedId="fieldValue(field)"
              :mode="'ENVIRONMENT'"
              :environmentId="environmentId"
              @select-database-id="
                (databaseId) => {
                  trySaveCustomField(field, databaseId);
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
      v-if="!$props.new"
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-6 grid-cols-3"
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

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Updated
      </h2>
      <span class="textfield col-span-2">
        {{ moment(task.lastUpdatedTs).format("LLL") }}</span
      >

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Created
      </h2>
      <span class="textfield col-span-2">
        {{ moment(task.createdTs).format("LLL") }}</span
      >
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
import PrincipalSelect from "../components/PrincipalSelect.vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import {
  TaskField,
  TaskBuiltinFieldId,
  DatabaseFieldPayload,
} from "../plugins";
import { DatabaseId, EnvironmentId, Task } from "../types";

interface LocalState {}

export default {
  name: "TaskSidebar",
  emits: ["update-assignee-id", "update-custom-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
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

    const environmentId = computed(() => {
      return props.task.payload[TaskBuiltinFieldId.ENVIRONMENT];
    });

    const allowEditAssignee = computed(() => {
      // We allow the current assignee or DBA/Owner to re-assign the task.
      // Though only DBA/Owner can be assigned to the task, the current
      // assignee might not have DBA/Owner role in case its role is revoked after
      // being assigned to the task.
      return (
        props.new ||
        (props.task.status == "OPEN" &&
          (currentUser.value.id == props.task.assignee?.id ||
            currentUser.value.role == "DBA" ||
            currentUser.value.role == "OWNER"))
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

    return {
      state,
      allowEditAssignee,
      fieldValue,
      environmentId,
      trySaveCustomField,
      trySaveDatabaseNew,
      trySaveDatabaseName,
      trySaveDatabaseId,
      trySaveDatabaseReadOnly,
    };
  },
};
</script>
