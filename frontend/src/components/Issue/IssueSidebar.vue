<template>
  <aside class="pr-0.5">
    <h2 class="sr-only">Details</h2>
    <div class="grid gap-y-6 gap-x-1 grid-cols-3">
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

      <template v-if="!create">
        <IssueReviewSidebarSection />
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1 gap-x-1">
        <span>{{ $t("common.assignee") }}</span>
        <span>
          <NTooltip>
            <template #trigger>
              <heroicons-outline:question-mark-circle />
            </template>
            <div>{{ $t("issue.assignee-tooltip") }}</div>
          </NTooltip>
        </span>
        <span v-if="create" class="text-red-600">*</span>
        <AssigneeAttentionButton />
      </h2>
      <!-- Only DBA can be assigned to the issue -->
      <div class="col-span-2" data-label="bb-assignee-select-container">
        <MemberSelect
          class="w-full"
          :disabled="!allowEditAssignee"
          :selected-id="assigneeId"
          :custom-filter="filterUser"
          data-label="bb-assignee-select"
          @select-user-id="updateAssigneeId"
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
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-1 grid-cols-3"
    >
      <template v-if="showStageSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.stage") }}
        </h2>
        <div class="col-span-2">
          <StageSelect
            :pipeline="(issue as Issue).pipeline!"
            :selected-id="(selectedStage as Stage).id as number"
            @select-stage-id="(stageId: number) => selectStageOrTask(stageId)"
          />
        </div>
      </template>

      <template v-if="showTaskSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.task") }}
        </h2>
        <div class="col-span-2">
          <TaskSelect
            :pipeline="(issue as Issue).pipeline!"
            :stage="(selectedStage as Stage)"
            :selected-id="(selectedTask as Task).id"
            @select-task-id="(taskId: IdType) => selectTaskId(taskId)"
          />
        </div>
      </template>

      <TaskRollbackView />

      <template v-if="!isTenantMode">
        <!--
          earliest-allowed-time is disabled in tenant mode for now
          we will provide more powerful deployment schedule in deployment config
         -->
        <div>
          <h2 class="textlabel flex items-center">
            <span class="mr-1">{{ $t("common.when") }}</span>
            <NTooltip>
              <template #trigger>
                <heroicons-outline:question-mark-circle class="h-4 w-4" />
              </template>
              <div class="w-60">
                {{ $t("task.earliest-allowed-time-hint") }}
              </div>
            </NTooltip>
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
          class="col-span-2 text-sm font-medium text-main truncate"
          :class="isDatabaseCreated ? 'cursor-pointer hover:underline' : ''"
          @click.prevent="
            {
              if (isDatabaseCreated) {
                clickDatabase();
              }
            }
          "
        >
          <div class="flex items-center gap-x-1">
            <span>{{ databaseName }}</span>
            <router-link
              :to="`/environment/${environmentV1Slug(environment)}`"
              class="col-span-2 text-sm font-medium text-main hover:underline"
            >
              ({{ environmentV1Name(environment) }})
            </router-link>
            <SQLEditorButtonV1
              v-if="databaseEntity"
              :database="databaseEntity"
            />
          </div>
          <div class="text-control-light">{{ showDatabaseCreationLabel }}</div>
        </div>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        <span class="mr-1">{{ $t("common.instance") }}</span>
        <InstanceV1EngineIcon :instance="instance" />
      </h2>
      <div class="flex gap-x-1">
        <router-link
          v-if="allowManageInstance"
          :to="`/instance/${instanceV1Slug(instance)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          {{ instanceV1Name(instance) }}
        </router-link>
        <span v-else class="col-span-2 text-sm font-medium text-main">
          {{ instanceV1Name(instance) }}
        </span>
        <router-link
          :to="`/environment/${environmentV1Slug(instance.environmentEntity)}`"
          class="col-span-2 text-sm font-medium text-main hover:underline"
        >
          ({{ environmentV1Name(instance.environmentEntity) }})
        </router-link>
      </div>

      <template v-for="label in visibleLabelList" :key="label.key">
        <h2
          class="textlabel flex items-start col-span-1 col-start-1 capitalize"
        >
          {{ hidePrefix(label.key) }}
        </h2>

        <div class="col-span-2 text-sm font-medium text-main">
          {{ label.value }}
        </div>
      </template>
    </div>
    <div
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-1 grid-cols-3"
    >
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.project") }}
      </h2>
      <ProjectV1Name
        :project="project"
        :link="true"
        :plain="true"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      />

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
      :open="state.showFeatureModal"
      :feature="'bb.feature.task-schedule-time'"
      :instance="database?.instanceEntity"
      @cancel="state.showFeatureModal = false"
    />
  </aside>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import { isEqual } from "lodash-es";
import { NDatePicker, NTooltip } from "naive-ui";
import { computed, PropType, reactive, ref, watch, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import { InstanceV1EngineIcon } from "@/components/v2";
import { ProjectV1Name } from "@/components/v2";
import { InputField } from "@/plugins";
import {
  featureToRef,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import {
  ComposedDatabase,
  Issue,
  IssueCreate,
  Task,
  TaskId,
  Stage,
  StageCreate,
  UNKNOWN_ID,
  ComposedInstance,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  allTaskList,
  hasWorkspacePermissionV1,
  hidePrefix,
  taskSlug,
  extractDatabaseNameFromTask,
  PRESET_LABEL_KEYS,
  extractUserUID,
  instanceV1Slug,
  instanceV1Name,
  environmentV1Slug,
  environmentV1Name,
  databaseV1Slug,
} from "@/utils";
import MemberSelect from "../MemberSelect.vue";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";
import IssueSubscriberPanel from "./IssueSubscriberPanel.vue";
import StageSelect from "./StageSelect.vue";
import TaskSelect from "./TaskSelect.vue";
import {
  allowUserToBeAssignee,
  allowUserToChangeAssignee,
  useCurrentRollOutPolicyForActiveEnvironment,
  useExtraIssueLogic,
  useIssueLogic,
} from "./logic";
import { IssueReviewSidebarSection } from "./review";
import TaskRollbackView from "./rollback/TaskRollbackView.vue";

dayjs.extend(isSameOrAfter);

interface LocalState {
  earliestAllowedTs: number | null;
  showFeatureModal: boolean;
}

const props = defineProps({
  database: {
    type: Object as PropType<ComposedDatabase | undefined>,
    default: undefined,
  },
  instance: {
    required: true,
    type: Object as PropType<ComposedInstance>,
  },
});

const router = useRouter();
const projectV1Store = useProjectV1Store();

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
  const currentUserUID = extractUserUID(currentUserV1.value.name);
  return (
    issueEntity.status === "OPEN" &&
    String(issueEntity.assignee?.id) === currentUserUID
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

const currentUserV1 = useCurrentUserV1();

const fieldValue = <T = string>(field: InputField): T => {
  return (issue.value.payload as Record<string, any>)[field.id] as T;
};

const databaseName = computed((): string | undefined => {
  return extractDatabaseNameFromTask(selectedTask.value);
});

const environment = computed((): Environment => {
  const environmentId = create.value
    ? (selectedStage.value as StageCreate).environmentId
    : (selectedStage.value as Stage).environment.id;

  return useEnvironmentV1Store().getEnvironmentByUID(String(environmentId));
});

const project = computed(() => {
  const projectUID = create.value
    ? (issue.value as IssueCreate).projectId
    : (issue.value as Issue).project.id;
  return projectV1Store.getProjectByUID(String(projectUID));
});

const assigneeId = computed(() => {
  if (create.value) {
    return String((issue.value as IssueCreate).assigneeId);
  }
  return String((issue.value as Issue).assignee.id);
});

const databaseEntity = ref<ComposedDatabase>();

const visibleLabelList = computed(() => {
  // transform non-reserved labels to db properties
  if (!props.database) return [];

  const labelList: { key: string; value: string }[] = [];
  for (const key in props.database.labels) {
    if (PRESET_LABEL_KEYS.includes(key)) {
      const value = props.database.labels[key];
      labelList.push({ key, value });
    }
  }
  return labelList;
});

const showStageSelect = computed((): boolean => {
  return (
    !create.value && allTaskList((issue.value as Issue).pipeline!).length > 1
  );
});

const showTaskSelect = computed((): boolean => {
  if (create.value) {
    return false;
  }
  const { taskList } = selectedStage.value;
  return taskList.length > 1;
});

const allowManageInstance = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});

const allowEditAssignee = computed(() => {
  if (create.value) {
    return true;
  }
  return allowUserToChangeAssignee(currentUserV1.value, issue.value as Issue);
});

const allowEditEarliestAllowedTime = computed(() => {
  if (create.value) {
    return true;
  }
  // only the assignee is allowed to modify EarliestAllowedTime
  const issueEntity = issue.value as Issue;
  const task = selectedTask.value as Task;
  const currentUserUID = extractUserUID(currentUserV1.value.name);
  return (
    issueEntity.status === "OPEN" &&
    (task.status === "PENDING" || task.status === "PENDING_APPROVAL") &&
    currentUserUID === String(issueEntity.assignee.id)
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
        databaseSlug: databaseV1Slug(props.database),
      },
    });
  } else {
    useDatabaseV1Store()
      .getOrFetchDatabaseByName(
        `${props.instance.name}/databases/${databaseName.value!}`
      )
      .then((database) => {
        router.push({
          name: "workspace.database.detail",
          params: {
            databaseSlug: databaseV1Slug(database),
          },
        });
      });
  }
};

watchEffect(() => {
  if (props.database) {
    databaseEntity.value = props.database;
  } else {
    const name = databaseName.value;
    if (name) {
      const existed = useDatabaseV1Store().getDatabaseByName(
        `${props.instance.name}/databases/${name}`
      );
      if (existed && existed.uid !== String(UNKNOWN_ID)) {
        databaseEntity.value = existed;
      } else {
        databaseEntity.value = undefined;
      }
    } else {
      databaseEntity.value = undefined;
    }
  }
});

const isDayPassed = (ts: number) => !dayjs(ts).isSameOrAfter(now, "day");

const hasInstanceFeature = featureToRef(
  "bb.feature.task-schedule-time",
  props.database?.instanceEntity
);

const updateEarliestAllowedTs = (newTimestampMS: number) => {
  if (!hasInstanceFeature.value) {
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
  selectStageOrTask(stage.id as number, slug);
};
const rollOutPolicy = useCurrentRollOutPolicyForActiveEnvironment();
const filterUser = (user: User): boolean => {
  return allowUserToBeAssignee(
    user,
    project.value,
    project.value.iamPolicy,
    rollOutPolicy.value.policy,
    rollOutPolicy.value.assigneeGroup
  );
};
</script>
