<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-3">
      <div
        v-if="!state.new"
        class="flex flex-row space-x-2 lg:flex-col lg:space-x-0"
      >
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Status</h2>
        <div class="lg:mt-3 w-3/4 lg:w-auto">
          <TaskStatusSelect
            :disabled="activeStageIsRunning(task)"
            :selectedStatus="task.attributes.status"
            @change-status="
              (value) => {
                $emit('update-task-status', value);
              }
            "
          />
        </div>
      </div>
      <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Assignee</h2>
        <ul class="lg:mt-3 w-3/4 lg:w-auto">
          <li class="flex justify-start items-center space-x-2">
            <div v-if="task.attributes.assignee" class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="task.attributes.assignee.name"
              />
            </div>
            <div class="text-sm font-medium text-main">
              {{
                task.attributes.assignee
                  ? task.attributes.assignee.name
                  : "Unassigned"
              }}
            </div>
          </li>
        </ul>
      </div>
      <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Reporter</h2>
        <ul class="lg:mt-3 w-3/4 lg:w-auto">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="task.attributes.creator.name"
              />
            </div>
            <div class="text-sm font-medium text-main">
              {{ task.attributes.creator.name }}
            </div>
          </li>
        </ul>
      </div>
      <template v-for="(field, index) in fieldList" :key="index">
        <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
          <h2 class="flex items-center textlabel w-1/4 lg:w-auto">
            {{ field.name }}
            <span v-if="field.required" class="text-red-600">*</span>
          </h2>
          <template v-if="field.type == 'String'">
            <div
              class="lg:mt-3 w-3/4 lg:w-auto mt-1 flex"
              @focusin="state.activeCustomFieldIndex = index"
              @focusout="
                (e) => {
                  // If we lose focus because of clicking the save/cancel button,
                  // we should NOT reset the active index. Otherwise, the button
                  // will be removed from the DOM before firing the click event.
                  if (
                    state.activeCustomFieldIndex == index &&
                    e.relatedTarget !== customFieldSaveButton &&
                    e.relatedTarget !== customFieldCancelButton
                  ) {
                    trySaveCustomTextField(index);
                  }
                }
              "
            >
              <input
                type="text"
                autocomplete="off"
                class="z-10 flex-1 min-w-0 block w-full border border-r border-control-border focus:ring-control focus:border-control sm:text-sm"
                :class="
                  state.activeCustomFieldIndex === index
                    ? 'rounded-l-md'
                    : 'rounded-md'
                "
                :ref="customFieldRefList[index]"
                :name="field.id"
                :value="fieldValue(field)"
                :placeholder="field.placeholder"
              />
              <template v-if="state.activeCustomFieldIndex === index">
                <button
                  tabindex="-1"
                  class="z-0 -ml-px px-1 border border-control-border text-sm font-medium text-control-light bg-control-bg hover:bg-control-bg-hover focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
                  ref="customFieldSaveButton"
                  @click="trySaveCustomTextField(index)"
                >
                  <svg
                    class="w-6 h-6 text-success"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M5 13l4 4L19 7"
                    ></path>
                  </svg>
                </button>
                <button
                  tabindex="-1"
                  class="z-0 -ml-px px-1 border border-control-border text-sm font-medium rounded-r-md text-control-light bg-control-bg hover:bg-control-bg-hover focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
                  ref="customFieldCancelButton"
                  @click="state.activeCustomFieldIndex = -1"
                >
                  <svg
                    class="w-6 h-6 text-control"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M6 18L18 6M6 6l12 12"
                    ></path>
                  </svg>
                </button>
              </template>
            </div>
          </template>
          <template v-else-if="field.type == 'Environment'">
            <div class="lg:mt-3 w-3/4 lg:w-auto">
              <EnvironmentSelect
                :name="field.id"
                :selectedId="fieldValue(field)"
                :selectDefault="false"
                @select-environment-id="
                  (environmentId) => {
                    $emit('update-custom-field', field, environmentId);
                  }
                "
              />
            </div>
          </template>
        </div>
      </template>
    </div>
    <div
      v-if="!state.new"
      class="mt-6 border-t border-block-border py-6 space-y-4"
    >
      <div>
        <h2 class="textlabel">Update Time</h2>
        <span class="textfield">
          {{ moment(task.attributes.lastUpdatedTs).format("LLL") }}</span
        >
      </div>
      <div>
        <h2 class="textlabel">Creation Time</h2>
        <span class="textfield">
          {{ moment(task.attributes.createdTs).format("LLL") }}</span
        >
      </div>
    </div>
  </aside>
</template>

<script lang="ts">
import {
  PropType,
  Ref,
  onMounted,
  onUnmounted,
  nextTick,
  reactive,
  ref,
  watchEffect,
} from "vue";
import isEmpty from "lodash-es/isEmpty";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import TaskStatusSelect from "../components/TaskStatusSelect.vue";
import { TaskField } from "../plugins";
import { Task } from "../types";
import { activeStageIsRunning } from "../utils";

interface LocalState {
  new: boolean;
  activeCustomFieldIndex: number;
}

export default {
  name: "TaskSidebar",
  emits: ["update-task-status", "update-custom-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    fieldList: {
      required: true,
      type: Object as PropType<TaskField[]>,
    },
  },
  components: { EnvironmentSelect, TaskStatusSelect },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      new: isEmpty(props.task.id),
      activeCustomFieldIndex: -1,
    });

    const customFieldSaveButton = ref();
    const customFieldCancelButton = ref();
    const customFieldRefList: Ref<HTMLElement | undefined>[] = [];
    for (const _ of props.fieldList) {
      customFieldRefList.push(ref());
    }

    const keyboardHandler = (e: KeyboardEvent) => {
      if (state.activeCustomFieldIndex != -1) {
        if (e.code == "Escape") {
          const index = state.activeCustomFieldIndex;
          state.activeCustomFieldIndex = -1;
          nextTick(() =>
            (customFieldRefList[index].value as HTMLElement).blur()
          );
        } else if (e.code == "Enter") {
          nextTick(() =>
            (customFieldRefList[state.activeCustomFieldIndex]
              .value as HTMLElement).blur()
          );
        }
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const refreshState = () => {
      state.new = isEmpty(props.task.id);
    };

    const fieldValue = (field: TaskField): string => {
      return field.preprocessor
        ? field.preprocessor(props.task.attributes.payload[field.id])
        : props.task.attributes.payload[field.id];
    };

    const trySaveCustomTextField = (customFieldIndex: number) => {
      const field = props.fieldList[customFieldIndex];
      let value = "";
      if (field.type === "String") {
        value = (customFieldRefList[customFieldIndex].value as HTMLInputElement)
          .value;
      } else if (field.type === "Environment") {
        value = (customFieldRefList[customFieldIndex]
          .value as HTMLSelectElement).value;
      }

      if (field.required && isEmpty(value)) {
        // Refocus
        nextTick(() =>
          (customFieldRefList[customFieldIndex].value as HTMLElement).focus()
        );
        return;
      }

      if (value != fieldValue(field)) {
        emit("update-custom-field", field, value);
      }

      state.activeCustomFieldIndex = -1;
    };

    watchEffect(refreshState);

    return {
      state,
      customFieldSaveButton,
      customFieldCancelButton,
      customFieldRefList,
      activeStageIsRunning,
      fieldValue,
      trySaveCustomTextField,
    };
  },
};
</script>
