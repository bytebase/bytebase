<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="grid gap-y-6 gap-x-6 grid-cols-3">
      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.status") }}
        </h2>
        <div class="col-span-2">
          <span class="flex items-center space-x-2">
            <IssueStatusIcon
              :issue-status="(issue as Issue).status"
              :size="'normal'"
            />
            <span class="text-main capitalize">
              {{ (issue as Issue).status.toLowerCase() }}
            </span>
          </span>
        </div>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.assignee")
        }}<span v-if="create" class="text-red-600">*</span>
      </h2>
      <!-- Only DBA can be assigned to the issue -->
      <div class="col-span-2">
        <!-- eslint-disable vue/attribute-hyphenation -->
        <MemberSelect
          class="w-full"
          :disabled="!allowEditAssignee"
          :selectedId="create ? (issue as IssueCreate).assigneeId : (issue as Issue).assignee?.id"
          :allowed-role-list="['OWNER', 'DBA']"
          @select-principal-id="
            (principalId: number) => {
              updateAssigneeId(principalId)
            }
          "
        />
      </div>

      <template v-for="(field, index) in template.inputFieldList" :key="index">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ field.name }}
          <span v-if="field.required" class="text-red-600">*</span>
        </h2>
        <div class="col-span-2">
          <template v-if="field.type == 'String'">
            <BBTextField
              class="text-sm"
              :disabled="!allowEditCustomField(field)"
              :required="true"
              :value="fieldValue(field)"
              :placeholder="field.placeholder"
              @end-editing="(text: string) => trySaveCustomField(field, text)"
            />
          </template>
          <template v-else-if="field.type == 'Boolean'">
            <BBSwitch
              :disabled="!allowEditCustomField(field)"
              :value="fieldValue(field)"
              @toggle="
                (on: boolean) => {
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
      <template v-if="showStageSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.stage") }}
        </h2>
        <div class="col-span-2">
          <StageSelect
            :pipeline="(issue as Issue).pipeline"
            :selected-id="(selectedStage as Stage).id"
            @select-stage-id="(stageId) => selectStageOrTask(stageId)"
          />
        </div>
      </template>

      <template v-if="showTaskSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.task") }}
        </h2>
        <div class="col-span-2">
          <TaskSelect
            :pipeline="(issue as Issue).pipeline"
            :stage="(selectedStage as Stage)"
            :selected-id="(selectedTask as Task).id"
            @select-task-id="(taskId) => selectTaskId(taskId)"
          />
        </div>
      </template>

      <template v-if="!isTenantMode">
        <!--
          earliest-allowed-time is disabled in tenant mode for now
          we will provide more powerful deployment schedule in deployment config
         -->
        <div>
          <h2 class="textlabel flex items-center">
            <span class="mr-1">{{ $t("common.when") }}</span>
            <div class="tooltip-wrapper">
              <span class="tooltip w-60">{{
                $t("task.earliest-allowed-time-hint")
              }}</span>
              <!-- Heroicons name: outline/question-mark-circle -->
              <heroicons-outline:question-mark-circle class="h-4 w-4" />
            </div>
          </h2>
          <h2 class="text-gray-600 text-sm">
            <span class="row-span-1">{{ "UTC" + dayjs().format("ZZ") }}</span>
          </h2>
        </div>

        <div class="col-span-2">
          <n-date-picker
            v-if="allowEditEarliestAllowedTime"
            :value="
              state.earliestAllowedTs ? state.earliestAllowedTs * 1000 : null
            "
            :is-date-disabled="isDayPassed"
            :placeholder="$t('task.earliest-allowed-time-unset')"
            class="w-full"
            type="datetime"
            clearable
            @update:value="updateEarliestAllowedTs"
          />

          <div v-else class="tooltip-wrapper">
            <span class="tooltip w-48 textlabel">{{
              $t("task.earliest-allowed-time-no-modify")
            }}</span>
            <span class="textfield col-span-2 text-sm font-medium text-main">
              {{
                selectedTask.earliestAllowedTs === 0
                  ? $t("task.earliest-allowed-time-unset")
                  : dayjs(selectedTask.earliestAllowedTs * 1000).format("LLL")
              }}
            </span>
          </div>
        </div>
      </template>

      <template v-if="databaseName">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.database") }}
        </h2>
        <div
          class="col-span-2 text-sm font-medium text-main"
          :class="isDatabaseCreated ? 'cursor-pointer hover:underline' : ''"
          @click.prevent="
            {
              if (isDatabaseCreated) {
                clickDatabase();
              }
            }
          "
        >
          {{ databaseName }}
          <span class="text-control-light">{{
            showDatabaseCreationLabel
          }}</span>
        </div>
      </template>

      <template v-if="showInstance">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          <span class="mr-1">{{ $t("common.instance") }}</span>
          <InstanceEngineIcon :instance="instance" />
        </h2>
        <router-link
          :to="`/instance/${instanceSlug(instance)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          {{ instanceName(instance) }}
        </router-link>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.environment") }}
      </h2>
      <router-link
        :to="`/environment/${environmentSlug(environment)}`"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      >
        {{ environmentName(environment) }}
      </router-link>

      <template v-for="label in visibleLabelList" :key="label.key">
        <h2
          class="textlabel flex items-start col-span-1 col-start-1 capitalize"
        >
          {{ hidePrefix(label.key) }}
        </h2>

        <div class="col-span-2 text-sm font-medium text-main capitalize">
          {{ label.value }}
        </div>
      </template>
    </div>
    <div
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-6 grid-cols-3"
    >
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.project") }}
      </h2>
      <router-link
        :to="`/project/${projectSlug(project)}`"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      >
        {{ projectName(project) }}
      </router-link>

      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.updated-at") }}
        </h2>
        <span class="textfield col-span-2">
          {{ dayjs((issue as Issue).updatedTs * 1000).format("LLL") }}</span
        >

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.created-at") }}
        </h2>
        <span class="textfield col-span-2">
          {{ dayjs((issue as Issue).createdTs * 1000).format("LLL") }}</span
        >
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.creator") }}
        </h2>
        <ul class="col-span-2">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <PrincipalAvatar
                :principal="(issue as Issue).creator"
                :size="'SMALL'"
              />
            </div>
            <router-link
              :to="`/u/${(issue as Issue).creator.id}`"
              class="text-sm font-medium text-main hover:underline"
            >
              {{ (issue as Issue).creator.name }}
            </router-link>
          </li>
        </ul>
      </template>
    </div>
    <IssueSubscriberPanel
      v-if="!create"
      :issue="(issue as Issue)"
      @add-subscriber-id="(subscriberId) => addSubscriberId(subscriberId)"
      @remove-subscriber-id="(subscriberId) => removeSubscriberId(subscriberId)"
    />
    <FeatureModal
      v-if="state.showFeatureModal"
      :feature="'bb.feature.task-schedule-time'"
      @cancel="state.showFeatureModal = false"
    />
  </aside>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { isEqual } from "lodash-es";
import { NDatePicker } from "naive-ui";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import StageSelect from "./StageSelect.vue";
import TaskSelect from "./TaskSelect.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";
import IssueSubscriberPanel from "./IssueSubscriberPanel.vue";
import InstanceEngineIcon from "../InstanceEngineIcon.vue";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import MemberSelect from "../MemberSelect.vue";
import FeatureModal from "../FeatureModal.vue";
import { InputField } from "@/plugins";
import type {
  Database,
  Environment,
  Project,
  Issue,
  IssueCreate,
  Task,
  TaskId,
  Stage,
  StageCreate,
  Instance,
  TaskDatabaseCreatePayload,
  DatabaseLabel,
} from "@/types";
import { ONBOARDING_ISSUE_ID } from "@/types";
import {
  allTaskList,
  databaseSlug,
  isDBAOrOwner,
  isReservedDatabaseLabel,
  hidePrefix,
  taskSlug,
} from "@/utils";
import {
  hasFeature,
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentStore,
  useLabelList,
  useProjectStore,
} from "@/store";
import { useExtraIssueLogic, useIssueLogic } from "./logic";

dayjs.extend(isSameOrAfter);

interface LocalState {
  earliestAllowedTs: number | null;
  showFeatureModal: boolean;
}

const props = defineProps({
  database: {
    type: Object as PropType<Database | undefined>,
    default: undefined,
  },
  instance: {
    required: true,
    type: Object as PropType<Instance>,
  },
});

const router = useRouter();
const projectStore = useProjectStore();

const {
  create,
  issue,
  template,
  isTenantMode,
  selectedStage,
  selectedTask,
  selectStageOrTask,
} = useIssueLogic();
const {
  updateEarliestAllowedTime,
  updateAssigneeId,
  updateCustomField,
  addSubscriberId,
  removeSubscriberId,
} = useExtraIssueLogic();

const allowEdit = computed(() => {
  if (create.value) {
    return true;
  }
  // For now, we only allow assignee to update the field when the issue
  // is 'OPEN'. This reduces flexibility as creator must ask assignee to
  // change any fields if there is typo. On the other hand, this avoids
  // the trouble that the creator changes field value when the creator
  // is performing the issue based on the old value.
  // For now, we choose to be on the safe side at the cost of flexibility.
  const issueEntity = issue.value as Issue;
  return (
    issueEntity.status == "OPEN" &&
    issueEntity.assignee?.id == currentUser.value.id
  );
});

const now = new Date();
const state = reactive<LocalState>({
  earliestAllowedTs: selectedTask.value.earliestAllowedTs,
  showFeatureModal: false,
});

watch(selectedTask, (cur) => {
  // we show user local time
  state.earliestAllowedTs = cur.earliestAllowedTs;
});

const currentUser = useCurrentUser();

const fieldValue = <T = string>(field: InputField): T => {
  return issue.value.payload[field.id] as T;
};

const databaseName = computed((): string | undefined => {
  if (props.database) {
    return props.database.name;
  }

  const stage = selectedStage.value as Stage;
  if (
    stage.taskList[0].type == "bb.task.database.create" ||
    stage.taskList[0].type == "bb.task.database.restore"
  ) {
    if (create.value) {
      const stage = selectedStage.value as StageCreate;
      return stage.taskList[0].databaseName;
    }
    return ((stage.taskList[0] as Task).payload as TaskDatabaseCreatePayload)
      .databaseName;
  }
  return undefined;
});

const environment = computed((): Environment => {
  if (create.value) {
    const stage = selectedStage.value as StageCreate;
    return useEnvironmentStore().getEnvironmentById(stage.environmentId);
  }
  const stage = selectedStage.value as Stage;
  return stage.environment;
});

const project = computed((): Project => {
  if (create.value) {
    return projectStore.getProjectById((issue.value as IssueCreate).projectId);
  }
  return (issue.value as Issue).project;
});

const labelList = useLabelList();

const visibleLabelList = computed((): DatabaseLabel[] => {
  // transform non-reserved labels to db properties
  if (!props.database) return [];
  if (labelList.value.length === 0) return [];

  return props.database.labels.filter(
    (label) => !isReservedDatabaseLabel(label, labelList.value)
  );
});

const showStageSelect = computed((): boolean => {
  return (
    !create.value && allTaskList((issue.value as Issue).pipeline).length > 1
  );
});

const showTaskSelect = computed((): boolean => {
  if (create.value) {
    return false;
  }
  const { taskList } = selectedStage.value;
  return taskList.length > 1;
});

const showInstance = computed((): boolean => {
  return isDBAOrOwner(currentUser.value.role);
});

const allowEditAssignee = computed(() => {
  if (create.value) {
    return true;
  }
  // We allow the current assignee or DBA to re-assign the issue.
  // Though only DBA can be assigned to the issue, the current
  // assignee might not have DBA role in case its role is revoked after
  // being assigned to the issue.
  const issueEntity = issue.value as Issue;
  return (
    issueEntity.id !== ONBOARDING_ISSUE_ID &&
    issueEntity.status == "OPEN" &&
    (currentUser.value.id == issueEntity.assignee.id ||
      isDBAOrOwner(currentUser.value.role))
  );
});

const allowEditEarliestAllowedTime = computed(() => {
  if (create.value) {
    return true;
  }
  // only the assignee is allowed to modify EarliestAllowedTime
  const issueEntity = issue.value as Issue;
  const task = selectedTask.value as Task;
  return (
    issueEntity.id != ONBOARDING_ISSUE_ID &&
    issueEntity.status == "OPEN" &&
    (task.status == "PENDING" || task.status == "PENDING_APPROVAL") &&
    currentUser.value.id == issueEntity.assignee.id
  );
});

const allowEditCustomField = (field: InputField) => {
  return allowEdit.value && (create.value || field.allowEditAfterCreation);
};

const trySaveCustomField = (field: InputField, value: string | boolean) => {
  if (!isEqual(value, fieldValue(field))) {
    updateCustomField(field, value);
  }
};

const isDatabaseCreated = computed(() => {
  const stage = selectedStage.value as Stage;
  if (stage.taskList[0].type == "bb.task.database.create") {
    if (create.value) {
      return false;
    }
    return stage.taskList[0].status == "DONE";
  }
  return true;
});

// We only show creation label for database create task
const showDatabaseCreationLabel = computed(() => {
  const stage = selectedStage.value as Stage;
  if (stage.taskList[0].type !== "bb.task.database.create") {
    return "";
  }
  return isDatabaseCreated.value ? "(created)" : "(pending create)";
});

const clickDatabase = () => {
  // If the database has not been created yet, do nothing
  if (props.database) {
    router.push({
      name: "workspace.database.detail",
      params: {
        databaseSlug: databaseSlug(props.database),
      },
    });
  } else {
    useDatabaseStore()
      .fetchDatabaseByInstanceIdAndName({
        instanceId: props.instance.id,
        name: databaseName.value!, // guarded in template to ensure databaseName is not empty
      })
      .then((database: Database) => {
        router.push({
          name: "workspace.database.detail",
          params: {
            databaseSlug: databaseSlug(database),
          },
        });
      });
  }
};

const isDayPassed = (ts: number) => !dayjs(ts).isSameOrAfter(now, "day");

const updateEarliestAllowedTs = (newTimestampMS: number) => {
  if (!hasFeature("bb.feature.task-schedule-time")) {
    state.showFeatureModal = true;
    return;
  }

  // n-date-picker would pass timestamp in millisecond.
  // We divide it by 1000 to get timestamp in second
  const newTs = newTimestampMS / 1000;
  state.earliestAllowedTs = newTs;
  updateEarliestAllowedTime(newTs);
};

const selectTaskId = (taskId: TaskId) => {
  const taskList = (selectedStage.value as Stage).taskList;
  const task = taskList.find((t) => t.id === taskId);
  if (!task) return;
  const slug = taskSlug(task.name, task.id);
  const stage = selectedStage.value as Stage;
  selectStageOrTask(stage.id, slug);
};
</script>
