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
            <IssueStatusIcon :issue-status="issue.status" :size="'normal'" />
            <span class="text-main capitalize">
              {{ issue.status.toLowerCase() }}
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
          :disabled="!allowEditAssignee"
          :selectedId="create ? issue.assigneeId : issue.assignee?.id"
          :allowed-role-list="['OWNER', 'DBA']"
          @select-principal-id="
            (principalId) => {
              $emit('update-assignee-id', principalId);
            }
          "
        />
      </div>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.when") }}
        <div class="tooltip-wrapper">
          <span class="tooltip w-60">{{
            $t("task.earliest-allowed-time-hint")
          }}</span>
          <!-- Heroicons name: outline/question-mark-circle -->
          <heroicons-outline:question-mark-circle class="h-4 w-4" />
        </div>
      </h2>
      <div class="col-span-2">
        <n-date-picker
          v-if="allowEditEarliestAllowedTime"
          v-model:value="state.earliestAllowedTs"
          :is-date-disabled="isDayPassed"
          :placeholder="$t('task.earliest-allowed-time-unset')"
          class="w-full"
          type="datetime"
          clearable
          @update:value="
            (newTimestampNs) => {
              $emit('update-earliest-allowed-time', newTimestampNs / 1000);
            }
          "
        />
        <span v-else class="textfield col-span-2">
          {{ moment(task.earliestAllowedTs * 1000).format("LLL") }}</span
        >
      </div>

      <template v-for="(field, index) in inputFieldList" :key="index">
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
              @end-editing="(text) => trySaveCustomField(field, text)"
            />
          </template>
          <template v-else-if="field.type == 'Boolean'">
            <BBSwitch
              :disabled="!allowEditCustomField(field)"
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
      <template v-if="showStageSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.stage") }}
        </h2>
        <div class="col-span-2">
          <StageSelect
            :pipeline="issue.pipeline"
            :selected-i-d="selectedStage.id"
            @select-stage-id="(stageId) => $emit('select-stage-id', stageId)"
          />
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
          {{ moment(issue.updatedTs * 1000).format("LLL") }}</span
        >

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.created-at") }}
        </h2>
        <span class="textfield col-span-2">
          {{ moment(issue.createdTs * 1000).format("LLL") }}</span
        >
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.creator") }}
        </h2>
        <ul class="col-span-2">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <PrincipalAvatar :principal="issue.creator" :size="'SMALL'" />
            </div>
            <router-link
              :to="`/u/${issue.creator.id}`"
              class="text-sm font-medium text-main hover:underline"
            >
              {{ issue.creator.name }}
            </router-link>
          </li>
        </ul>
      </template>
    </div>
    <IssueSubscriberPanel
      v-if="!create"
      :issue="issue"
      @add-subscriber-id="
        (subscriberId) => $emit('add-subscriber-id', subscriberId)
      "
      @remove-subscriber-id="
        (subscriberId) => $emit('remove-subscriber-id', subscriberId)
      "
    />
  </aside>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useStore } from "vuex";
import isEqual from "lodash-es/isEqual";
import { NDatePicker } from "naive-ui";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import MemberSelect from "../components/MemberSelect.vue";
import StageSelect from "../components/StageSelect.vue";
import IssueStatusIcon from "../components/IssueStatusIcon.vue";
import IssueSubscriberPanel from "../components/IssueSubscriberPanel.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";

import { InputField } from "../plugins";
import {
  Database,
  Environment,
  Project,
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  EMPTY_ID,
  Stage,
  StageCreate,
  Instance,
  ONBOARDING_ISSUE_ID,
  TaskDatabaseCreatePayload,
} from "../types";
import { allTaskList, databaseSlug, isDBAOrOwner } from "../utils";
import { useRouter } from "vue-router";
import moment from "moment";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "IssueSidebar",
  components: {
    NDatePicker,
    PrincipalAvatar,
    MemberSelect,
    StageSelect,
    IssueStatusIcon,
    IssueSubscriberPanel,
    InstanceEngineIcon,
  },
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue | IssueCreate>,
    },
    task: {
      required: true,
      type: Object as PropType<Task | TaskCreate>,
    },
    create: {
      required: true,
      type: Boolean,
    },
    selectedStage: {
      required: true,
      type: Object as PropType<Stage | StageCreate>,
    },
    inputFieldList: {
      required: true,
      type: Object as PropType<InputField[]>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
    database: {
      required: true,
      type: Object as PropType<Database | undefined>,
    },
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
  },
  emits: [
    "update-assignee-id",
    "update-earliest-allowed-time",
    "add-subscriber-id",
    "remove-subscriber-id",
    "update-custom-field",
    "select-stage-id",
  ],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const now = new Date();
    const state = reactive<LocalState>({
      earliestAllowedTs: props.task.earliestAllowedTs
        ? props.task.earliestAllowedTs * 1000
        : null,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const fieldValue = (field: InputField): string => {
      return props.issue.payload[field.id];
    };

    const databaseName = computed((): string | undefined => {
      if (props.database) {
        return props.database.name;
      }

      const stage = props.selectedStage as Stage;
      if (
        stage.taskList[0].type == "bb.task.database.create" ||
        stage.taskList[0].type == "bb.task.database.restore"
      ) {
        if (props.create) {
          const stage = props.selectedStage as StageCreate;
          return stage.taskList[0].databaseName;
        }
        return (
          (stage.taskList[0] as Task).payload as TaskDatabaseCreatePayload
        ).databaseName;
      }
      return undefined;
    });

    const environment = computed((): Environment => {
      if (props.create) {
        const stage = props.selectedStage as StageCreate;
        return store.getters["environment/environmentById"](
          stage.environmentId
        );
      }
      const stage = props.selectedStage as Stage;
      return stage.environment;
    });

    const project = computed((): Project => {
      if (props.create) {
        return store.getters["project/projectById"](
          (props.issue as IssueCreate).projectId
        );
      }
      return (props.issue as Issue).project;
    });

    const showStageSelect = computed((): boolean => {
      return (
        !props.create && allTaskList((props.issue as Issue).pipeline).length > 1
      );
    });

    const showInstance = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowEditAssignee = computed(() => {
      // We allow the current assignee or DBA to re-assign the issue.
      // Though only DBA can be assigned to the issue, the current
      // assignee might not have DBA role in case its role is revoked after
      // being assigned to the issue.
      return (
        props.create ||
        ((props.issue as Issue).id != ONBOARDING_ISSUE_ID &&
          (props.issue as Issue).status == "OPEN" &&
          (currentUser.value.id == (props.issue as Issue).assignee?.id ||
            isDBAOrOwner(currentUser.value.role)))
      );
    });

    const allowEditEarliestAllowedTime = computed(() => {
      // only the assignee is allowed to modify EarliestAllowedTime
      return (
        props.create ||
        ((props.issue as Issue).id != ONBOARDING_ISSUE_ID &&
          (props.issue as Issue).status == "OPEN" &&
          currentUser.value.id == (props.issue as Issue).assignee?.id)
      );
    });

    const allowEditCustomField = (field: InputField) => {
      return props.allowEdit && (props.create || field.allowEditAfterCreation);
    };

    const trySaveCustomField = (field: InputField, value: string | boolean) => {
      if (!isEqual(value, fieldValue(field))) {
        emit("update-custom-field", field, value);
      }
    };

    const isDatabaseCreated = computed(() => {
      const stage = props.selectedStage as Stage;
      if (stage.taskList[0].type == "bb.task.database.create") {
        if (props.create) {
          return false;
        }
        return stage.taskList[0].status == "DONE";
      }
      return true;
    });

    // We only show creation label for database create task
    const showDatabaseCreationLabel = computed(() => {
      const stage = props.selectedStage as Stage;
      if (stage.taskList[0].type != "bb.task.database.create") {
        return "";
      }
      return isDatabaseCreated.value ? "(created)" : "(pending create)";
    });

    // TODO: errors detected by Vetur below is related to https://github.com/bytebase/bytebase/issues/56
    // Will fix this in another branch.
    const clickDatabase = () => {
      // If the database has not been created yet, do nothing
      if (props.database && props.database.value) {
        router.push({
          name: "workspace.database.detail",
          params: {
            databaseSlug: databaseSlug(props.database.value),
          },
        });
      } else {
        store
          .dispatch("database/fetchDatabaseByInstanceIdAndName", {
            instanceId: props.instance.id,
            name: databaseName.value,
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

    const isDayPassed = (ts: number) => !moment(ts).isSameOrAfter(now, "day");

    return {
      EMPTY_ID,
      state,
      fieldValue,
      environment,
      databaseName,
      project,
      showInstance,
      showStageSelect,
      allowEditAssignee,
      allowEditEarliestAllowedTime,
      allowEditCustomField,
      trySaveCustomField,
      isDatabaseCreated,
      showDatabaseCreationLabel,
      clickDatabase,
      isDayPassed,
    };
  },
});
</script>
