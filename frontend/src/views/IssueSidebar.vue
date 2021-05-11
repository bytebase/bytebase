<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="grid gap-y-6 gap-x-6 grid-cols-3">
      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Status
        </h2>
        <div class="col-span-2">
          <span class="flex items-center space-x-2">
            <IssueStatusIcon :issueStatus="issue.status" :size="'normal'" />
            <span class="text-main capitalize">
              {{ issue.status.toLowerCase() }}
            </span>
          </span>
        </div>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Assignee<span v-if="create" class="text-red-600">*</span>
      </h2>
      <!-- Only DBA can be assigned to the issue -->
      <div class="col-span-2">
        <PrincipalSelect
          :disabled="!allowEditAssignee"
          :selectedId="create ? issue.assigneeId : issue.assignee?.id"
          :allowedRoleList="['OWNER', 'DBA']"
          @select-principal-id="
            (principalId) => {
              $emit('update-assignee-id', principalId);
            }
          "
        />
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
      class="
        mt-6
        border-t border-block-border
        pt-6
        grid
        gap-y-6 gap-x-6
        grid-cols-3
      "
    >
      <template v-if="showStageSelect">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Stage
        </h2>
        <div class="col-span-2">
          <StageSelect
            :pipeline="issue.pipeline"
            :selectedId="selectedStage.id"
            @select-stage-id="(stageId) => $emit('select-stage-id', stageId)"
          />
        </div>
      </template>
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Database
      </h2>
      <router-link
        :to="`/db/${databaseSlug(database)}`"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      >
        {{ database.id == EMPTY_ID ? "N/A" : database.name }}
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
    </div>
    <div
      class="
        mt-6
        border-t border-block-border
        pt-6
        grid
        gap-y-6 gap-x-6
        grid-cols-3
      "
    >
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        Project
      </h2>
      <router-link
        :to="`/project/${projectSlug(project)}`"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      >
        {{ projectName(project) }}
      </router-link>

      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Updated
        </h2>
        <span class="textfield col-span-2">
          {{ moment(issue.updatedTs).format("LLL") }}</span
        >

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Created
        </h2>
        <span class="textfield col-span-2">
          {{ moment(issue.createdTs).format("LLL") }}</span
        >
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          Creator
        </h2>
        <ul class="col-span-2">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar :size="'small'" :username="issue.creator.name" />
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
    <div
      v-if="!create"
      class="
        mt-6
        border-t border-block-border
        pt-6
        grid
        gap-y-4 gap-x-6
        grid-cols-3
      "
    >
      <h2
        class="
          textlabel
          flex
          items-center
          col-span-1 col-start-1
          whitespace-nowrap
        "
      >
        {{
          issue.subscriberList.length +
          (issue.subscriberList.length > 1 ? " subscribers" : " subscriber")
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
import StageSelect from "../components/StageSelect.vue";
import IssueStatusIcon from "../components/IssueStatusIcon.vue";
import { InputField } from "../plugins";
import {
  Database,
  Environment,
  Principal,
  Project,
  Issue,
  IssueCreate,
  EMPTY_ID,
  Stage,
} from "../types";
import { allTaskList, isDBAOrOwner } from "../utils";
import { useRouter } from "vue-router";

interface LocalState {}

export default {
  name: "IssueSidebar",
  emits: [
    "update-assignee-id",
    "update-subscriber-list",
    "update-custom-field",
    "select-stage-id",
  ],
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue | IssueCreate>,
    },
    create: {
      required: true,
      type: Boolean,
    },
    selectedStage: {
      required: true,
      type: Object as PropType<Stage>,
    },
    inputFieldList: {
      required: true,
      type: Object as PropType<InputField[]>,
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
    StageSelect,
    IssueStatusIcon,
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const fieldValue = (field: InputField): string => {
      return props.issue.payload[field.id];
    };

    const database = computed((): Database => {
      if (props.create) {
        const databaseId = (props.issue as IssueCreate).pipeline?.stageList[0]
          .taskList[0].databaseId;
        return store.getters["database/databaseById"](databaseId);
      }
      const stage = props.selectedStage;
      return stage.taskList[0].database;
    });

    const environment = computed((): Environment => {
      if (props.create) {
        const environmentId = (props.issue as IssueCreate).pipeline
          ?.stageList[0].environmentId;
        return store.getters["environment/environmentById"](environmentId);
      }
      const stage = props.selectedStage;
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

    const isCurrentUserSubscribed = computed((): boolean => {
      for (const principal of (props.issue as Issue).subscriberList) {
        if (currentUser.value.id == principal.id) {
          return true;
        }
      }
      return false;
    });

    const subscriberList = computed((): Principal[] => {
      const list: Principal[] = [];
      (props.issue as Issue).subscriberList.forEach((principal: Principal) => {
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
      // We allow the current assignee or DBA to re-assign the issue.
      // Though only DBA can be assigned to the issue, the current
      // assignee might not have DBA role in case its role is revoked after
      // being assigned to the issue.
      return (
        props.create ||
        ((props.issue as Issue).status == "OPEN" &&
          (currentUser.value.id == (props.issue as Issue).assignee?.id ||
            isDBAOrOwner(currentUser.value.role)))
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

    const toggleSubscription = () => {
      const list = cloneDeep((props.issue as Issue).subscriberList);
      if (isCurrentUserSubscribed.value) {
        const index = (props.issue as Issue).subscriberList.findIndex(
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
      EMPTY_ID,
      state,
      fieldValue,
      environment,
      database,
      project,
      showStageSelect,
      isCurrentUserSubscribed,
      subscriberList,
      allowEditAssignee,
      allowEditCustomField,
      trySaveCustomField,
      toggleSubscription,
    };
  },
};
</script>
