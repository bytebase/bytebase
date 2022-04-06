<template>
  <div>
    <div class="divide-y divide-block-border">
      <div class="pb-4">
        <h2 id="activity-title" class="text-lg font-medium text-main">
          {{ $t("common.activity") }}
        </h2>
      </div>
      <div class="pt-6">
        <!-- Activity feed-->
        <ul>
          <li v-for="(activity, index) in activityList" :key="index">
            <div :id="'activity' + activity.id" class="relative pb-4">
              <span
                v-if="index != activityList.length - 1"
                class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
                aria-hidden="true"
              ></span>
              <div class="relative flex items-start">
                <template v-if="actionIcon(activity) == 'system'">
                  <div class="relative">
                    <div class="relative pl-0.5">
                      <div
                        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                      >
                        <img
                          class="mt-1"
                          src="../../assets/logo-icon.svg"
                          alt="Bytebase"
                        />
                      </div>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'avatar'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <PrincipalAvatar
                        :principal="activity.creator"
                        :size="'SMALL'"
                      />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'create'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-solid:plus-sm class="w-5 h-5 text-control" />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'update'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-solid:pencil class="w-4 h-4 text-control" />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'run'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:play class="w-6 h-6 text-control" />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'approve'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:thumb-up
                        class="w-5 h-5 text-control"
                      />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'cancel'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:minus class="w-5 h-5 text-control" />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'fail'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:exclamation-circle
                        class="w-6 h-6 text-error"
                      />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'complete'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:check-circle
                        class="w-6 h-6 text-success"
                      />
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'commit'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
                    >
                      <heroicons-outline:code class="w-5 h-5 text-control" />
                    </div>
                  </div>
                </template>
                <div class="ml-3 min-w-0 flex-1">
                  <div class="min-w-0 flex-1 pt-1 flex justify-between">
                    <div class="text-sm text-control-light">
                      {{ actionSubjectPrefix(activity) }}
                      <router-link
                        :to="actionSubject(activity).link"
                        class="font-medium text-main whitespace-nowrap hover:underline"
                        exact-active-class=""
                        >{{ actionSubject(activity).name }}</router-link
                      >
                      <a
                        :href="'#activity' + activity.id"
                        class="ml-1 anchor-link whitespace-normal"
                      >
                        {{ actionSentence(activity) }}
                        {{ humanizeTs(activity.createdTs) }}
                        <template
                          v-if="
                            activity.createdTs != activity.updatedTs &&
                            activity.type == 'bb.issue.comment.create'
                          "
                        >
                          ({{ $t("common.edited") }}
                          {{ humanizeTs(activity.updatedTs) }})
                        </template>
                      </a>
                    </div>
                    <div
                      v-if="currentUser.id == activity.creator.id"
                      class="space-x-2 flex items-center text-control-light"
                    >
                      <template
                        v-if="
                          state.editCommentMode &&
                          state.activeActivity.id == activity.id
                        "
                      >
                        <button
                          type="button"
                          class="rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                          @click.prevent="cancelEditComment"
                        >
                          {{ $t("common.cancel") }}
                        </button>
                        <button
                          type="button"
                          class="border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                          :disabled="!allowUpdateComment"
                          @click.prevent="doUpdateComment"
                        >
                          {{ $t("common.save") }}
                        </button>
                      </template>
                      <!-- mr-2 is to vertical align with the text description edit button-->
                      <div v-else class="mr-2 flex items-center space-x-2">
                        <!-- Delete Comment Button-->
                        <button
                          v-if="activity.type == 'bb.issue.comment.create'"
                          class="btn-icon"
                          @click.prevent="
                            {
                              state.activeActivity = activity;
                              state.showDeleteCommentModal = true;
                            }
                          "
                        >
                          <heroicons-outline:trash class="w-4 h-4" />
                        </button>
                        <!-- Edit Comment Button-->
                        <button
                          class="btn-icon"
                          @click.prevent="onUpdateComment(activity)"
                        >
                          <heroicons-outline:pencil class="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  <div class="mt-2 text-sm text-control whitespace-pre-wrap">
                    <template
                      v-if="
                        state.editCommentMode &&
                        state.activeActivity.id == activity.id
                      "
                    >
                      <label for="comment" class="sr-only">
                        {{ $t("issue.edit-comment") }}
                      </label>
                      <textarea
                        ref="editCommentTextArea"
                        v-model="editComment"
                        rows="3"
                        class="textarea block w-full resize-none"
                        :placeholder="$t('issue.leave-a-comment')"
                        @input="
                          (e) => {
                            sizeToFit(e.target);
                          }
                        "
                        @focus="
                          (e) => {
                            sizeToFit(e.target);
                          }
                        "
                      ></textarea>
                    </template>
                    <template v-else>
                      {{ activity.comment }}
                    </template>
                    <template
                      v-if="activity.type == 'bb.pipeline.task.file.commit'"
                    >
                      <a
                        :href="`${activity.payload.vcsInstanceUrl}/${activity.payload.repositoryFullPath}/-/commit/${activity.payload.commitId}`"
                        target="__blank"
                        class="normal-link flex flex-row items-center"
                      >
                        {{ $t("issue.view-commit") }}
                        <heroicons-outline:external-link class="w-4 h-4" />
                      </a>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </li>
        </ul>

        <div v-if="!state.editCommentMode">
          <div class="flex">
            <div class="flex-shrink-0">
              <div class="relative">
                <PrincipalAvatar :principal="currentUser" />
                <span
                  class="absolute -bottom-0.5 -right-1 bg-white rounded-tl px-0.5 py-px"
                >
                  <!-- Heroicon name: solid/chat-alt -->
                  <heroicons-solid:chat-alt
                    class="h-3.5 w-3.5 text-control-light"
                  />
                </span>
              </div>
            </div>
            <div class="ml-3 min-w-0 flex-1">
              <label for="comment" class="sr-only">
                {{ $t("issue.add-a-comment") }}
              </label>
              <textarea
                ref="newCommentTextArea"
                v-model="newComment"
                rows="3"
                class="textarea block w-full resize-none whitespace-pre-wrap"
                :placeholder="$t('issue.leave-a-comment')"
                @input="
                  (e) => {
                    sizeToFit(e.target);
                  }
                "
              ></textarea>
              <div class="mt-4 flex items-center justify-start space-x-4">
                <button
                  type="button"
                  class="btn-normal"
                  :disabled="newComment.length == 0"
                  @click.prevent="doCreateComment"
                >
                  {{ $t("common.comment") }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <BBAlert
    v-if="state.showDeleteCommentModal"
    :style="'INFO'"
    :ok-text="'Delete'"
    :title="'Are you sure to delete this comment?'"
    :description="'You cannot undo this action.'"
    @ok="
      () => {
        doDeleteComment(state.activeActivity);
        state.showDeleteCommentModal = false;
        state.activeActivity = null;
      }
    "
    @cancel="state.showDeleteCommentModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import {
  onMounted,
  onUnmounted,
  computed,
  nextTick,
  ref,
  reactive,
  watchEffect,
  PropType,
  defineComponent,
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import {
  Issue,
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
  UNKNOWN_ID,
  EMPTY_ID,
  SYSTEM_BOT_ID,
  ActivityCreate,
  IssueSubscriber,
  ActivityTaskFileCommitPayload,
} from "../../types";
import {
  findTaskById,
  issueActivityActionSentence,
  issueSlug,
  sizeToFit,
  taskSlug,
} from "../../utils";
import { IssueTemplate, IssueBuiltinFieldId } from "../../plugins";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import { useCurrentUser, useUIStateStore } from "@/store";

interface LocalState {
  showDeleteCommentModal: boolean;
  editCommentMode: boolean;
  activeActivity?: Activity;
}

interface ActionSubject {
  name: string;
  link: string;
}

type ActionIconType =
  | "avatar"
  | "system"
  | "create"
  | "update"
  | "run"
  | "approve"
  | "cancel"
  | "fail"
  | "complete"
  | "commit";

export default defineComponent({
  name: "IssueActivityPanel",
  components: { PrincipalAvatar },
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    issueTemplate: {
      required: true,
      type: Object as PropType<IssueTemplate>,
    },
  },
  emits: ["add-subscriber-id"],
  setup(props, { emit }) {
    const { t } = useI18n();
    const store = useStore();
    const router = useRouter();

    const newComment = ref("");
    const newCommentTextArea = ref();
    const editComment = ref("");
    const editCommentTextArea = ref();

    const state = reactive<LocalState>({
      showDeleteCommentModal: false,
      editCommentMode: false,
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (
        state.editCommentMode &&
        editCommentTextArea.value === document.activeElement
      ) {
        if (e.code == "Escape") {
          cancelEditComment();
        } else if (e.code == "Enter" && e.metaKey) {
          if (allowUpdateComment.value) {
            doUpdateComment();
          }
        }
      } else if (newCommentTextArea.value === document.activeElement) {
        if (e.code == "Enter" && e.metaKey) {
          doCreateComment();
        }
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const currentUser = useCurrentUser();

    const prepareActivityList = () => {
      store.dispatch("activity/fetchActivityListForIssue", props.issue.id);
    };

    watchEffect(prepareActivityList);

    // The activity list and its anchor is not immediately available when the issue shows up.
    // Thus the scrollBehavior set in the vue router won't work (also tried promise to resolve async with no luck either)
    // So we manually use scrollIntoView after rendering the activity list.
    nextTick(() => {
      if (router.currentRoute.value.hash) {
        const el = document.getElementById(
          router.currentRoute.value.hash.slice(1)
        );
        el?.scrollIntoView();
      }
    });

    // Need to use computed to make list reactive to activity list changes.
    const activityList = computed((): Activity[] => {
      const list = store.getters["activity/activityListByIssue"](
        props.issue.id
      );
      return list.filter((activity: Activity) => {
        if (activity.type == "bb.issue.field.update") {
          let containUserVisibleChange =
            (activity.payload as ActivityIssueFieldUpdatePayload).fieldId !=
            IssueBuiltinFieldId.SUBSCRIBER_LIST;
          return containUserVisibleChange;
        }
        return true;
      });
    });

    const subscriberList = computed((): IssueSubscriber[] => {
      return store.getters["issueSubscriber/subscriberListByIssue"](
        props.issue.id
      );
    });

    const cancelEditComment = () => {
      editComment.value = "";
      state.activeActivity = undefined;
      state.editCommentMode = false;
    };

    const doCreateComment = () => {
      const createActivity: ActivityCreate = {
        type: "bb.issue.comment.create",
        containerId: props.issue.id,
        comment: newComment.value,
      };
      store.dispatch("activity/createActivity", createActivity).then(() => {
        useUIStateStore().saveIntroStateByKey({
          key: "comment.create",
          newState: true,
        });

        newComment.value = "";
        nextTick(() => sizeToFit(newCommentTextArea.value));

        // Because the user just added a comment and we assume she is interested in this
        // issue, and we add her to the subscriber list if she is not there
        let isSubscribed = false;
        for (const subscriber of subscriberList.value) {
          if (subscriber.subscriber.id == currentUser.value.id) {
            isSubscribed = true;
            break;
          }
        }
        if (!isSubscribed) {
          emit("add-subscriber-id", currentUser.value.id);
        }
      });
    };

    const onUpdateComment = (activity: Activity) => {
      editComment.value = activity.comment;
      state.activeActivity = activity;
      state.editCommentMode = true;
      nextTick(() => {
        editCommentTextArea.value.focus();
      });
    };

    const doUpdateComment = () => {
      store
        .dispatch("activity/updateComment", {
          activityId: state.activeActivity!.id,
          updatedComment: editComment.value,
        })
        .then(() => {
          cancelEditComment();
        });
    };

    const allowUpdateComment = computed(() => {
      return editComment.value != state.activeActivity!.comment;
    });

    const doDeleteComment = (activity: Activity) => {
      store.dispatch("activity/deleteActivity", activity);
    };

    const actionIcon = (activity: Activity): ActionIconType => {
      if (activity.type == "bb.issue.create") {
        return "create";
      } else if (activity.type == "bb.issue.field.update") {
        return "update";
      } else if (activity.type == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActivityTaskStatusUpdatePayload;
        switch (payload.newStatus) {
          case "PENDING": {
            if (payload.oldStatus == "RUNNING") {
              return "cancel";
            } else if (payload.oldStatus == "PENDING_APPROVAL") {
              return "approve";
            }
            break;
          }
          case "RUNNING": {
            return "run";
          }
          case "DONE": {
            return "complete";
          }
          case "FAILED": {
            return "fail";
          }
        }
      } else if (activity.type == "bb.pipeline.task.file.commit") {
        return "commit";
      } else if (activity.type == "bb.pipeline.task.statement.update") {
        return "update";
      } else if (
        activity.type == "bb.pipeline.task.general.earliest-allowed-time.update"
      ) {
        return "update";
      }

      return activity.creator.id == SYSTEM_BOT_ID ? "system" : "avatar";
    };

    const actionSubjectPrefix = (activity: Activity): string => {
      if (activity.creator.id == SYSTEM_BOT_ID) {
        if (activity.type == "bb.pipeline.task.status.update") {
          return `${t("activity.subject-prefix.task")} `;
        }
      }
      return "";
    };

    const actionSubject = (activity: Activity): ActionSubject => {
      if (activity.creator.id == SYSTEM_BOT_ID) {
        if (activity.type == "bb.pipeline.task.status.update") {
          if (props.issue.pipeline.id != EMPTY_ID) {
            const payload = activity.payload as ActivityTaskStatusUpdatePayload;
            const task = findTaskById(props.issue.pipeline, payload.taskId);
            var link = "";
            if (task.id != UNKNOWN_ID) {
              link = `/issue/${issueSlug(
                props.issue.name,
                props.issue.id
              )}?task=${taskSlug(task.name, task.id)}`;
            }
            return {
              name: `${task.name} (${task.stage.name})`,
              link,
            };
          }
        }
      }
      return {
        name: activity.creator.name,
        link: `/u/${activity.creator.id}`,
      };
    };

    const actionSentence = (activity: Activity): string => {
      if (activity.type.startsWith("bb.issue.")) {
        const [tid, params] = issueActivityActionSentence(activity);
        return t(tid, params);
      }
      switch (activity.type) {
        case "bb.pipeline.task.status.update": {
          const payload = activity.payload as ActivityTaskStatusUpdatePayload;
          let str = t("activity.sentence.changed");
          switch (payload.newStatus) {
            case "PENDING": {
              if (payload.oldStatus == "RUNNING") {
                str = t("activity.sentence.canceled");
              } else if (payload.oldStatus == "PENDING_APPROVAL") {
                str = t("activity.sentence.approved");
              }
              break;
            }
            case "RUNNING": {
              str = t("activity.sentence.started");
              break;
            }
            case "DONE": {
              str = t("activity.sentence.completed");
              break;
            }
            case "FAILED": {
              str = t("activity.sentence.failed");
              break;
            }
          }
          if (activity.creator.id != SYSTEM_BOT_ID) {
            // If creator is not the robot (which means we do NOT use task name in the subject),
            // then we append the task name here.
            const task = findTaskById(props.issue.pipeline, payload.taskId);
            str += t("activity.sentence.task-name", { name: task.name });
          }
          return str;
        }
        case "bb.pipeline.task.file.commit": {
          const payload = activity.payload as ActivityTaskFileCommitPayload;
          // return `committed ${payload.filePath} to ${payload.branch}@${payload.repositoryFullPath}`;
          return t("activity.sentence.committed-to-at", {
            file: payload.filePath,
            branch: payload.branch,
            repo: payload.repositoryFullPath,
          });
        }
        case "bb.pipeline.task.statement.update": {
          const payload =
            activity.payload as ActivityTaskStatementUpdatePayload;
          return t("activity.sentence.changed-from-to", {
            name: "SQL",
            oldValue: payload.oldStatement,
            newValue: payload.newStatement,
          });
        }
        case "bb.pipeline.task.general.earliest-allowed-time.update": {
          const payload =
            activity.payload as ActivityTaskEarliestAllowedTimeUpdatePayload;
          const newVal = payload.newEarliestAllowedTs;
          const oldVal = payload.oldEarliestAllowedTs;
          return t("activity.sentence.changed-from-to", {
            name: "earliest allowed time",
            oldValue: oldVal ? dayjs(oldVal * 1000) : "Unset",
            newValue: newVal ? dayjs(newVal * 1000) : "Unset",
          });
        }
      }
      return "";
    };

    return {
      SYSTEM_BOT_ID,
      state,
      activityList,
      newComment,
      newCommentTextArea,
      editComment,
      editCommentTextArea,
      currentUser,
      actionIcon,
      actionSubjectPrefix,
      actionSubject,
      actionSentence,
      doCreateComment,
      cancelEditComment,
      onUpdateComment,
      doUpdateComment,
      allowUpdateComment,
      doDeleteComment,
    };
  },
});
</script>
