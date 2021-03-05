<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-6">
      <div v-if="!$props.new" class="flex flex-row space-x-2">
        <h2 class="flex items-center textlabel w-36">Status</h2>
        <div class="z-10 w-full">
          <TaskStatusSelect
            :disabled="activeStageIsRunning(task)"
            :selectedStatus="task.status"
            @change-status="
              (value) => {
                $emit('update-task-status', value);
              }
            "
          />
        </div>
      </div>
      <div class="flex flex-row space-x-2">
        <h2 class="flex items-center textlabel w-36">Assignee</h2>
        <div class="w-full">
          <PrincipalSelect
            :selectedId="task.assignee?.id"
            @select-principal="
              (principal) => {
                $emit('update-assignee-id', principal.id);
              }
            "
          />
        </div>
      </div>
      <template v-for="(field, index) in fieldList" :key="index">
        <div class="flex flex-row space-x-2">
          <h2 class="flex items-center textlabel w-36">
            {{ field.name }}
            <span v-if="field.required" class="text-red-600">*</span>
          </h2>
          <template v-if="field.type == 'String'">
            <div
              class="w-full flex"
              @focusin="
                if (!$props.new) {
                  state.activeCustomFieldIndex = index;
                }
              "
              @focusout="
                (e) => {
                  // If we lose focus because of clicking the save/cancel button,
                  // we should NOT reset the active index. Otherwise, the button
                  // will be removed from the DOM before firing the click event.
                  if (
                    !$props.new &&
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
                class="flex-1 min-w-0 block w-full border border-r border-control-border focus:mr-0.5 focus:ring-control focus:border-control sm:text-sm"
                :class="
                  state.activeCustomFieldIndex === index
                    ? 'rounded-l-md'
                    : 'rounded-md'
                "
                :ref="customFieldRefList[index]"
                :name="field.id"
                :value="fieldValue(field)"
                :placeholder="field.placeholder"
                @input="
                  if ($props.new) {
                    trySaveCustomTextField(index);
                  }
                "
              />
              <template
                v-if="!$props.new && state.activeCustomFieldIndex === index"
              >
                <button
                  tabindex="-1"
                  class="-ml-px px-1 border border-control-border text-sm font-medium text-control-light bg-control-bg hover:bg-control-bg-hover focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
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
                  class="-ml-px px-1 border border-control-border text-sm font-medium rounded-r-md text-control-light bg-control-bg hover:bg-control-bg-hover focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
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
            <div class="w-full">
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
      v-if="!$props.new"
      class="mt-6 border-t border-block-border pt-6 space-y-6"
    >
      <div class="flex flex-row space-x-2">
        <h2 class="flex items-center textlabel w-36">Reporter</h2>
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
import {
  PropType,
  Ref,
  onMounted,
  onUnmounted,
  nextTick,
  reactive,
  ref,
} from "vue";
import isEmpty from "lodash-es/isEmpty";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import TaskStatusSelect from "../components/TaskStatusSelect.vue";
import { TaskField } from "../plugins";
import { Task } from "../types";
import { activeStageIsRunning } from "../utils";

interface LocalState {
  activeCustomFieldIndex: number;
}

export default {
  name: "TaskSidebar",
  emits: ["update-task-status", "update-assignee-id", "update-custom-field"],
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
  components: { EnvironmentSelect, PrincipalSelect, TaskStatusSelect },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
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

    const fieldValue = (field: TaskField): string => {
      return field.preprocessor
        ? field.preprocessor(props.task.payload[field.id])
        : props.task.payload[field.id];
    };

    const trySaveCustomTextField = (customFieldIndex: number) => {
      const field = props.fieldList[customFieldIndex];
      let value = "";
      if (field.type === "String") {
        const el = customFieldRefList[customFieldIndex]
          .value as HTMLInputElement;
        value = el.value;
        if (field.preprocessor) {
          value = field.preprocessor(value);
          el.value = value;
        }
      } else if (field.type === "Environment") {
        const el = customFieldRefList[customFieldIndex]
          .value as HTMLSelectElement;
        value = el.value;
        if (field.preprocessor) {
          value = field.preprocessor(value);
          el.value = value;
        }
      }

      if (!props.new && field.required && isEmpty(value)) {
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
