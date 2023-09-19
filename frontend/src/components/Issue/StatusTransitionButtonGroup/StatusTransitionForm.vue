<template>
  <div class="px-4 space-y-6 divide-y divide-block-border">
    <div class="mt-2 grid grid-cols-1 gap-x-4 sm:grid-cols-4">
      <template v-if="mode == 'ISSUE' && transition.type == 'RESOLVE'">
        <template v-for="(field, index) in outputFieldList" :key="index">
          <div class="flex flex-row items-center text-sm">
            <div class="sm:col-span-1">
              <label class="textlabel">
                {{ field.name }}
              </label>
            </div>
          </div>
          <div class="sm:col-span-4 sm:col-start-1">
            <template v-if="field.type === 'String'">
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
            <template v-if="field.type === 'Database'">
              <DatabaseSelect
                class="mt-1 w-64"
                :disabled="true"
                :mode="'ENVIRONMENT'"
                :environment-id="environmentId"
                :selected-id="state.outputValueList[index]"
                @select-database-id="state.outputValueList[index] = $event"
              />
            </template>
          </div>
          <div v-if="index == outputFieldList.length - 1" class="mt-4" />
        </template>
      </template>

      <div v-if="showTaskCheckBar" class="sm:col-span-4 mb-4 space-y-4">
        <template v-if="checkSummary.runningCount > 0">
          <BBAttention
            :style="'INFO'"
            :title="`${checkSummary.runningCount} check(s) in progress...`"
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

      <div v-if="showTaskList" class="sm:col-span-4 mb-4">
        <label for="about" class="textlabel">
          {{ $t("common.tasks") }}
        </label>
        <ul class="mt-1 max-h-[6rem] overflow-y-auto">
          <li
            v-for="item in distinctTaskList"
            :key="item.task.id"
            class="text-sm textinfolabel"
          >
            <span class="textinfolabel">
              {{ item.task.name }}
            </span>
            <span v-if="item.similar.length > 0" class="ml-2 text-gray-400">
              {{
                $t("task.n-similar-tasks", {
                  count: item.similar.length + 1,
                })
              }}
            </span>
          </li>
        </ul>
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
        {{ cancelText }}
      </button>
      <button
        type="button"
        class="ml-3 px-4 py-2"
        :class="submitButtonStyle"
        :disabled="!allowSubmit"
        @click.prevent="$emit('submit', state.comment)"
      >
        {{ displayingOkText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { cloneDeep, groupBy } from "lodash-es";
import { computed, reactive, ref, PropType, defineComponent } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseSelect from "@/components/DatabaseSelect.vue";
import { OutputField } from "@/plugins";
import { Issue, IssueStatusTransition, Task } from "@/types";
import {
  activeEnvironment,
  activeStage,
  taskCheckRunSummary,
  TaskStatusTransition,
} from "@/utils";
import TaskCheckBar from "../TaskCheckBar.vue";
import { useIssueLogic } from "../logic";

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
      type: String as PropType<"ISSUE" | "STAGE" | "TASK">,
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
    const { t } = useI18n();
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      comment: "",
      outputValueList: props.outputFieldList.map((field) =>
        cloneDeep((props.issue.payload as Record<string, string>)[field.id])
      ),
    });

    const { allowApplyTaskStatusTransition } = useIssueLogic();

    const checkSummary = computed(() => taskCheckRunSummary(props.task));

    const cancelText = computed(() => t("common.cancel"));
    const displayingOkText = computed(() => {
      if (props.okText === cancelText.value) {
        // We don't want to see [Cancel] [Cancel]
        // So fall back to [Cancel] [Confirm] if okText===cancelText
        return t("common.confirm");
      }
      return props.okText;
    });

    const environmentId = computed(() => {
      return String(activeEnvironment(props.issue.pipeline).id);
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
        case "STAGE": // fallthrough the same as TASK
        case "TASK": {
          switch ((props.transition as TaskStatusTransition).type) {
            case "RUN":
              return "btn-primary";
            case "ROLLOUT":
              return "btn-primary";
            case "RETRY":
              return "btn-primary";
            case "CANCEL":
              return "btn-danger";
            case "SKIP":
              return "btn-primary";
            case "RESTART":
              return "btn-normal";
          }
        }
      }
      return ""; // only to make eslint happy
    });

    const showTaskCheckBar = computed((): boolean => {
      if (props.mode !== "TASK") return false;
      const taskCheckCount = props.task?.taskCheckRunList.length ?? 0;
      return taskCheckCount > 0;
    });

    const showTaskList = computed((): boolean => {
      return props.mode === "STAGE";
    });

    const taskList = computed(() => {
      const stage = activeStage(props.issue.pipeline!);
      const transition = props.transition as TaskStatusTransition;
      return stage.taskList.filter((task) => {
        return allowApplyTaskStatusTransition(task, transition.to);
      });
    });

    const distinctTaskList = computed(() => {
      type DistinctTaskList = { task: Task; similar: Task[] };
      const groups = groupBy(taskList.value, (task) => task.name);

      return Object.keys(groups).map<DistinctTaskList>((taskName) => {
        const [task, ...similar] = groups[taskName];
        return { task, similar };
      });
    });

    // Code block below will raise an eslint ERROR.
    // But I won't change it this time.
    // Disable submit if in TASK mode and there exists RUNNING check or check error and we are transitioning to RUNNING
    const allowSubmit = computed(() => {
      switch (props.mode) {
        case "ISSUE": {
          return true;
        }
        case "STAGE": // fallthrough the same as TASK
        case "TASK": {
          switch ((props.transition as TaskStatusTransition).to) {
            case "RUNNING":
            case "PENDING": // fallthrough
              return (
                checkSummary.value.runningCount === 0 &&
                checkSummary.value.errorCount === 0
              );
            default:
              return true;
          }
        }
      }
      return false; // only to make eslint happy
    });

    return {
      state,
      checkSummary,
      cancelText,
      displayingOkText,
      environmentId,
      showTaskCheckBar,
      showTaskList,
      distinctTaskList,
      commentTextArea,
      submitButtonStyle,
      allowSubmit,
    };
  },
});
</script>
