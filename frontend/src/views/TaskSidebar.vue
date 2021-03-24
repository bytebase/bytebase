<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-6">
      <div v-if="!$props.new" class="flex flex-row space-x-2 pb-4 border-b">
        <h2 class="flex items-center textlabel w-36">Status</h2>
        <div class="w-full">
          <span class="flex items-center space-x-2">
            <TaskStatusIcon :taskStatus="task.status" :size="'normal'" />
            <span class="text-main capitalize">
              {{ task.status.toLowerCase() }}
            </span>
          </span>
        </div>
      </div>
      <div class="flex flex-row space-x-2">
        <h2 class="flex items-center textlabel w-36">
          Assignee<span v-if="$props.new" class="text-red-600">*</span>
        </h2>
        <div class="w-full">
          <PrincipalSelect
            :disabled="!allowEditAssignee"
            :selectedId="task.assignee?.id"
            :allowAllRoles="false"
            @select-principal-id="
              (principalId) => {
                $emit('update-assignee-id', principalId);
              }
            "
          />
        </div>
      </div>
      <template v-for="(field, index) in fieldList" :key="index">
        <div class="flex flex-row space-x-2">
          <template v-if="field.type == 'String'">
            <h2 class="flex items-center textlabel w-36">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <BBTextField
              class="w-full mt-4 text-sm"
              :required="true"
              :value="fieldValue(field)"
              :placeholder="field.placeholder"
              @end-editing="(text) => trySaveCustomField(field, text)"
            />
          </template>
          <template v-else-if="field.type == 'Environment'">
            <h2 class="flex items-center textlabel w-36">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <div class="w-full">
              <EnvironmentSelect
                :name="field.id"
                :selectedId="fieldValue(field)"
                :selectDefault="false"
                @select-environment-id="
                  (environmentId) => {
                    trySaveCustomField(field, environmentId);
                  }
                "
              />
            </div>
          </template>
          <template v-else-if="field.type == 'Database'">
            <h2 class="flex items-center textlabel w-36">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <div class="w-full">
              <DatabaseSelect
                :disabled="!environmentId()"
                :selectedId="fieldValue(field)"
                :environmentId="environmentId()"
                @select-database-id="
                  (databaseId) => {
                    trySaveCustomField(field, databaseId);
                  }
                "
              />
            </div>
          </template>
          <template v-else-if="field.type == 'NewDatabase'">
            <h2 class="flex textlabel mt-2 w-36">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <div class="flex flex-col w-full">
              <div class="flex flex-row space-x-2">
                <BBCheckbox
                  :label="'New'"
                  :value="fieldValue(field).isNew"
                  class="items-center"
                  @toggle="
                    (on) => {
                      trySaveDatabaseNew(field, on);
                    }
                  "
                />
                <BBTextField
                  v-if="fieldValue(field).isNew"
                  type="text"
                  class="w-full text-sm"
                  :required="true"
                  :value="fieldValue(field).name"
                  :placeholder="field.placeholder"
                  @end-editing="(text) => trySaveDatabaseName(field, text)"
                />
                <DatabaseSelect
                  v-else
                  :disabled="!environmentId()"
                  :selectedId="fieldValue(field).id"
                  :environmentId="environmentId()"
                  @select-database-id="
                    (databaseId) => {
                      trySaveDatabaseId(field, databaseId);
                    }
                  "
                />
              </div>
              <BBSwitch
                v-if="!fieldValue(field).isNew"
                class="mt-4 flex"
                style="margin-left: 3.75rem"
                :label="'Read only'"
                :value="fieldValue(field).readOnly"
                @toggle="
                  (on) => {
                    trySaveDatabaseReadOnly(field, on);
                  }
                "
              />
            </div>
          </template>
          <template v-else-if="field.type == 'Switch'">
            <h2 class="flex items-center textlabel w-36">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <div class="flex justify-start">
              <BBSwitch
                :value="fieldValue(field)"
                @toggle="
                  (on) => {
                    trySaveCustomField(field, on);
                  }
                "
              />
            </div>
          </template>
        </div>
      </template>
    </div>
    <div
      v-if="!$props.new"
      class="mt-6 border-t border-block-border pt-6 space-y-6"
    >
      <div class="flex flex-row space-x-2">
        <h2 class="flex items-center textlabel w-36">Creator</h2>
        <ul class="w-full">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar :size="'small'" :username="task.creator.name" />
            </div>
            <div class="text-sm font-medium text-main">
              {{ task.creator.name }}
            </div>
          </li>
        </ul>
      </div>
      <div class="flex flex-row space-x-2">
        <h2 class="textlabel w-36">Updated</h2>
        <span class="textfield w-full">
          {{ moment(task.lastUpdatedTs).format("LLL") }}</span
        >
      </div>
      <div class="flex flex-row space-x-2">
        <h2 class="textlabel w-36">Created</h2>
        <span class="textfield w-full">
          {{ moment(task.createdTs).format("LLL") }}</span
        >
      </div>
    </div>
  </aside>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
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
import { activeStageIsRunning } from "../utils";

interface LocalState {}

export default {
  name: "TaskSidebar",
  emits: [
    "start-status-transition",
    "update-assignee-id",
    "update-custom-field",
  ],
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
      // Do a deep clone to prevent caller accidentally changes the original data.
      return cloneDeep(props.task.payload[field.id]);
    };

    const environmentId = (): string => {
      return props.task.payload[TaskBuiltinFieldId.ENVIRONMENT];
    };

    const allowEditAssignee = computed(() => {
      // We allow the current assignee or DBA/Owner to re-assign the task.
      // Though only DBA/Owner can be assigned to the task, the current
      // assignee might not have DBA/Owner role in case its role is revoked after
      // being assigned to the task.
      return (
        props.new ||
        currentUser.value.id == props.task.assignee?.id ||
        currentUser.value.role == "DBA" ||
        currentUser.value.role == "OWNER"
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
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.isNew = isNew;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseName = (field: TaskField, value: string) => {
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.name = value;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseId = (field: TaskField, value: DatabaseId) => {
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.id = value;
      trySaveCustomField(field, payload);
    };

    const trySaveDatabaseReadOnly = (field: TaskField, value: boolean) => {
      const payload: DatabaseFieldPayload = cloneDeep(
        fieldValue(field)
      ) as DatabaseFieldPayload;
      payload.readOnly = value;
      trySaveCustomField(field, payload);
    };

    return {
      state,
      activeStageIsRunning,
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
