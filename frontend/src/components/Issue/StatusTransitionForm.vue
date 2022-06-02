<template>
  <div class="px-4 space-y-6 divide-y divide-block-border">
    <div class="mt-2 grid grid-cols-1 gap-x-4 sm:grid-cols-4">
      <template v-if="mode == 'ISSUE' && transition.type == 'RESOLVE'">
        <template v-for="(field, index) in outputFieldList" :key="index">
          <div class="flex flex-row items-center text-sm">
            <div class="sm:col-span-1">
              <label class="textlabel">
                {{ field.name }}
                <span v-if="field.required" class="text-red-600">*</span>
              </label>
            </div>
          </div>
          <div class="sm:col-span-4 sm:col-start-1">
            <template v-if="field.type == 'String'">
              <div class="mt-1 flex rounded-md shadow-sm">
                <input
                  :id="field.id"
                  v-model="state.outputValueList[index]"
                  type="text"
                  disabled="true"
                  :name="field.id"
                  autocomplete="off"
                  class="w-full textfield"
                />
              </div>
            </template>
            <template v-if="field.type == 'Database'">
              <!-- eslint-disable vue/attribute-hyphenation -->
              <DatabaseSelect
                class="mt-1 w-64"
                :disabled="true"
                :mode="'ENVIRONMENT'"
                :environmentId="environmentId"
                :selectedId="state.outputValueList[index]"
                @select-database-id="
                  (databaseId: string) => {
                    state.outputValueList[index] = databaseId;
                  }
                "
              />
            </template>
          </div>
          <div v-if="index == outputFieldList.length - 1" class="mt-4" />
        </template>
      </template>

      <div
        v-if="mode == 'TASK' && task.taskCheckRunList.length > 0"
        class="sm:col-span-4 mb-4 space-y-4"
      >
        <template v-if="runningCheckCount > 0">
          <BBAttention
            :style="'INFO'"
            :title="`${runningCheckCount} check(s) in progress...`"
          />
        </template>
        <template v-else>
          <BBAttention
            v-if="checkSummary.errorCount > 0"
            :style="'CRITICAL'"
            :title="
              allowSubmit
                ? `Check found ${checkSummary.errorCount} error(s) and ${checkSummary.warnCount} warning(s)`
                : `Check found ${checkSummary.errorCount} error(s) and ${checkSummary.warnCount} warning(s), please fix before proceeding`
            "
          />
          <BBAttention
            v-else-if="checkSummary.warnCount > 0"
            :style="'WARN'"
            :title="`Check found ${checkSummary.warnCount} warning(s)`"
          />
        </template>
        <TaskCheckBar :task="task!" :allow-run-task="false" />
      </div>

      <div class="sm:col-span-4 w-112 min-w-full">
        <label for="about" class="textlabel">
          {{ $t("issue.status-transition.form.note") }}
        </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            v-model="state.comment"
            rows="3"
            class="textarea block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
            :placeholder="$t('issue.status-transition.form.placeholder')"
            @input="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
            @focus="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
          ></textarea>
        </div>
      </div>
    </div>

    <!-- Update button group -->
    <div class="flex justify-end items-center pt-5">
      <button
        type="button"
        class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
        @click.prevent="$emit('cancel')"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="ml-3 px-4 py-2"
        :class="submitButtonStyle"
        :disabled="!allowSubmit"
        @click.prevent="$emit('submit', state.comment)"
      >
        {{ okText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, ref, PropType, defineComponent } from "vue";
import { cloneDeep, groupBy, maxBy } from "lodash-es";
import DatabaseSelect from "../DatabaseSelect.vue";
import TaskCheckBar from "./TaskCheckBar.vue";
import { Issue, IssueStatusTransition, Task } from "@/types";
import { OutputField } from "@/plugins";
import { activeEnvironment, TaskStatusTransition } from "@/utils";

type CheckSummary = {
  successCount: number;
  warnCount: number;
  errorCount: number;
};

interface LocalState {
  comment: string;
  outputValueList: string[];
}

export default defineComponent({
  name: "StatusTransitionForm",
  components: { DatabaseSelect, TaskCheckBar },
  props: {
    mode: {
      required: true,
      type: String as PropType<"ISSUE" | "TASK">,
    },
    okText: {
      type: String,
      default: "OK",
    },
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    // Applicable when mode = 'TASK'
    task: {
      type: Object as PropType<Task>,
      default: undefined,
    },
    transition: {
      required: true,
      type: Object as PropType<IssueStatusTransition | TaskStatusTransition>,
    },
    outputFieldList: {
      required: true,
      type: Object as PropType<OutputField[]>,
    },
  },
  emits: ["submit", "cancel"],
  setup(props) {
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      comment: "",
      outputValueList: props.outputFieldList.map((field) =>
        cloneDeep(props.issue.payload[field.id])
      ),
    });

    const environmentId = computed(() => {
      return activeEnvironment(props.issue.pipeline).id;
    });

    // Code block below will raise an eslint ERROR.
    // But I won't change it this time.
    const submitButtonStyle = computed(() => {
      switch (props.mode) {
        case "ISSUE": {
          switch ((props.transition as IssueStatusTransition).type) {
            case "RESOLVE":
              return "btn-success";
            case "CANCEL":
              return "btn-danger";
            case "REOPEN":
              return "btn-primary";
          }
          break; // only to make eslint happy
        }
        case "TASK": {
          switch ((props.transition as TaskStatusTransition).type) {
            case "RUN":
              return "btn-primary";
            case "APPROVE":
              return "btn-primary";
            case "RETRY":
              return "btn-primary";
            case "CANCEL":
              return "btn-danger";
            case "SKIP":
              return "btn-danger";
          }
        }
      }
      return ""; // only to make eslint happy
    });

    // Code block below will raise an eslint ERROR.
    // But I won't change it this time.
    // Disable submit if in TASK mode and there exists RUNNING check or check error and we are transitioning to RUNNING
    const allowSubmit = computed(() => {
      switch (props.mode) {
        case "ISSUE": {
          return true;
        }
        case "TASK": {
          switch ((props.transition as TaskStatusTransition).to) {
            case "RUNNING":
              return (
                runningCheckCount.value == 0 &&
                checkSummary.value.errorCount == 0
              );
            default:
              return true;
          }
        }
      }
      return false; // only to make eslint happy
    });

    const runningCheckCount = computed((): number => {
      let count = 0;
      for (const run of props.task!.taskCheckRunList) {
        if (run.status == "RUNNING") {
          count++;
        }
      }
      return count;
    });

    const checkSummary = computed((): CheckSummary => {
      const summary: CheckSummary = {
        successCount: 0,
        warnCount: 0,
        errorCount: 0,
      };

      const taskCheckRunList = props.task?.taskCheckRunList ?? [];

      const listGroupByType = groupBy(taskCheckRunList, (run) => run.type);
      const latestCheckRunOfEachType = Object.keys(listGroupByType).map(
        (type) => {
          const listOfType = listGroupByType[type];
          const latest = maxBy(listOfType, (run) => run.updatedTs)!;
          return latest;
        }
      );

      for (const check of latestCheckRunOfEachType) {
        if (check.status == "DONE") {
          for (const result of check.result.resultList) {
            if (result.status == "SUCCESS") {
              summary.successCount++;
            } else if (result.status == "WARN") {
              summary.warnCount++;
            } else if (result.status == "ERROR") {
              summary.errorCount++;
            }
          }
        } else if (check.status == "FAILED") {
          summary.errorCount++;
        }
      }
      return summary;
    });

    return {
      state,
      environmentId,
      commentTextArea,
      submitButtonStyle,
      allowSubmit,
      runningCheckCount,
      checkSummary,
    };
  },
});
</script>
