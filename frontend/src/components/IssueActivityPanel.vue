<template>
  <div>
    <div class="divide-y divide-block-border">
      <div class="pb-4">
        <h2 id="activity-title" class="text-lg font-medium text-main">
          Activity
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
                        class="
                          w-7
                          h-7
                          bg-control-bg
                          rounded-full
                          ring-4 ring-white
                          flex
                          items-center
                          justify-center
                        "
                      >
                        <img
                          class="mt-1"
                          src="../assets/logo-icon.svg"
                          alt="Bytebase"
                        />
                      </div>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'avatar'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-white
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
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
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-5 h-5 text-control"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          fill-rule="evenodd"
                          d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z"
                          clip-rule="evenodd"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'update'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-4 h-4 text-control"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'run'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
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
                          d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                        ></path>
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'approve'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-5 h-5 text-control"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M14 10h4.764a2 2 0 011.789 2.894l-3.5 7A2 2 0 0115.263 21h-4.017c-.163 0-.326-.02-.485-.06L7 20m7-10V5a2 2 0 00-2-2h-.095c-.5 0-.905.405-.905.905 0 .714-.211 1.412-.608 2.006L7 11v9m7-10h-2M7 20H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.5"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'cancel'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-5 h-5 text-control"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M20 12H4"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'fail'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-white
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-6 h-6 text-error"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'complete'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-white
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
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
                          d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <template v-else-if="actionIcon(activity) == 'commit'">
                  <div class="relative pl-0.5">
                    <div
                      class="
                        w-7
                        h-7
                        bg-control-bg
                        rounded-full
                        ring-4 ring-white
                        flex
                        items-center
                        justify-center
                      "
                    >
                      <svg
                        class="w-5 h-5 text-control"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
                        ></path>
                      </svg>
                    </div>
                  </div>
                </template>
                <div class="ml-3 min-w-0 flex-1">
                  <div class="min-w-0 flex-1 pt-1 flex justify-between">
                    <div class="text-sm text-control-light">
                      {{ actionSubjectPrefix(activity) }}
                      <router-link
                        :to="actionSubject(activity).link"
                        class="
                          font-medium
                          text-main
                          whitespace-nowrap
                          hover:underline
                        "
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
                          (edited
                          {{ humanizeTs(activity.createdTs) }})
                        </template>
                      </a>
                      <span
                        v-if="
                          activity.type == 'bb.issue.create' &&
                          activity.payload.rollbackIssueID
                        "
                      >
                        (rollback
                        <router-link
                          :to="`/issue/${activity.payload.rollbackIssueID}`"
                          class="normal-link"
                          >{{ `issue/${activity.payload.rollbackIssueID}` }}
                        </router-link>
                        )
                      </span>
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
                          class="
                            rounded-sm
                            text-control
                            hover:bg-control-bg-hover
                            disabled:bg-control-bg
                            disabled:opacity-50
                            disabled:cursor-not-allowed
                            px-2
                            text-xs
                            leading-5
                            font-normal
                            focus:ring-control focus:outline-none
                            focus-visible:ring-2
                            focus:ring-offset-2
                          "
                          @click.prevent="cancelEditComment"
                        >
                          Cancel
                        </button>
                        <button
                          type="button"
                          class="
                            border border-control-border
                            rounded-sm
                            text-control
                            bg-control-bg
                            hover:bg-control-bg-hover
                            disabled:bg-control-bg
                            disabled:opacity-50
                            disabled:cursor-not-allowed
                            px-2
                            text-xs
                            leading-5
                            font-normal
                            focus:ring-control focus:outline-none
                            focus-visible:ring-2
                            focus:ring-offset-2
                          "
                          :disabled="!allowUpdateComment"
                          @click.prevent="doUpdateComment"
                        >
                          Save
                        </button>
                      </template>
                      <!-- mr-2 is to veritical align with the text description edit button-->
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
                          <svg
                            class="w-4 h-4"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                            xmlns="http://www.w3.org/2000/svg"
                          >
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              stroke-width="2"
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            ></path>
                          </svg>
                        </button>
                        <!-- Edit Comment Button-->
                        <button
                          class="btn-icon"
                          @click.prevent="onUpdateComment(activity)"
                        >
                          <svg
                            class="w-4 h-4"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                            xmlns="http://www.w3.org/2000/svg"
                          >
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              stroke-width="2"
                              d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                            ></path>
                          </svg>
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
                      <label for="comment" class="sr-only">Edit Comment</label>
                      <textarea
                        ref="editCommentTextArea"
                        rows="3"
                        class="textarea block w-full resize-none"
                        placeholder="Leave a comment..."
                        v-model="editComment"
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
                        :href="`${activity.payload.vcsInstanceURL}/${activity.payload.repositoryFullPath}/-/commit/${activity.payload.commitID}`"
                        target="__blank"
                        class="normal-link flex flex-row items-center"
                        >View commit
                        <svg
                          class="w-4 h-4"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                          xmlns="http://www.w3.org/2000/svg"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                          ></path>
                        </svg>
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
                  class="
                    absolute
                    -bottom-0.5
                    -right-1
                    bg-white
                    rounded-tl
                    px-0.5
                    py-px
                  "
                >
                  <!-- Heroicon name: solid/chat-alt -->
                  <svg
                    class="h-3.5 w-3.5 text-control-light"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M18 5v8a2 2 0 01-2 2h-5l-5 4v-4H4a2 2 0 01-2-2V5a2 2 0 012-2h12a2 2 0 012 2zM7 8H5v2h2V8zm2 0h2v2H9V8zm6 0h-2v2h2V8z"
                      clip-rule="evenodd"
                    />
                  </svg>
                </span>
              </div>
            </div>
            <div class="ml-3 min-w-0 flex-1">
              <label for="comment" class="sr-only">Create Comment</label>
              <textarea
                ref="newCommentTextArea"
                rows="3"
                class="textarea block w-full resize-none whitespace-pre-wrap"
                placeholder="Leave a comment..."
                v-model="newComment"
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
                  Comment
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
    :okText="'Delete'"
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
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  Issue,
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  UNKNOWN_ID,
  EMPTY_ID,
  SYSTEM_BOT_ID,
  ActivityCreate,
  IssueSubscriber,
  ActivityTaskFileCommitPayload,
} from "../types";
import {
  findTaskByID,
  issueActivityActionSentence,
  issueSlug,
  sizeToFit,
  stageSlug,
  taskSlug,
} from "../utils";
import { IssueTemplate, IssueBuiltinFieldID } from "../plugins";

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

export default {
  name: "IssueActivityPanel",
  emits: ["add-subscriber-id"],
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
  components: { PrincipalAvatar },
  setup(props, { emit }) {
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

    const currentUser = computed(() => store.getters["auth/currentUser"]());

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
            (activity.payload as ActivityIssueFieldUpdatePayload).fieldID !=
            IssueBuiltinFieldID.SUBSCRIBER_LIST;
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
        containerID: props.issue.id,
        comment: newComment.value,
      };
      store.dispatch("activity/createActivity", createActivity).then(() => {
        store.dispatch("uistate/saveIntroStateByKey", {
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
      const activityPatch = store
        .dispatch("activity/updateComment", {
          activityID: state.activeActivity!.id,
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
      }

      return activity.creator.id == SYSTEM_BOT_ID ? "system" : "avatar";
    };

    const actionSubjectPrefix = (activity: Activity): string => {
      if (activity.creator.id == SYSTEM_BOT_ID) {
        if (activity.type == "bb.pipeline.task.status.update") {
          return "Task ";
        }
      }
      return "";
    };

    const actionSubject = (activity: Activity): ActionSubject => {
      if (activity.creator.id == SYSTEM_BOT_ID) {
        if (activity.type == "bb.pipeline.task.status.update") {
          if (props.issue.pipeline.id != EMPTY_ID) {
            const payload = activity.payload as ActivityTaskStatusUpdatePayload;
            const task = findTaskByID(props.issue.pipeline, payload.taskID);
            var link = "";
            if (task.id != UNKNOWN_ID) {
              link = `/issue/${issueSlug(
                props.issue.name,
                props.issue.id
              )}?task=${taskSlug(task)}`;
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
        return issueActivityActionSentence(activity);
      }
      switch (activity.type) {
        case "bb.pipeline.task.status.update": {
          const payload = activity.payload as ActivityTaskStatusUpdatePayload;
          var str = `changed`;
          switch (payload.newStatus) {
            case "PENDING": {
              if (payload.oldStatus == "RUNNING") {
                str = `canceled`;
              } else if (payload.oldStatus == "PENDING_APPROVAL") {
                str = `approved`;
              }
              break;
            }
            case "RUNNING": {
              str = `started`;
              break;
            }
            case "DONE": {
              str = `completed`;
              break;
            }
            case "FAILED": {
              str = `failed`;
              break;
            }
          }
          if (activity.creator.id != SYSTEM_BOT_ID) {
            // If creator is not the robot (which means we do NOT use task name in the subject),
            // then we append the task name here.
            const task = findTaskByID(props.issue.pipeline, payload.taskID);
            str += ` task ${task.name}`;
          }
          return str;
        }
        case "bb.pipeline.task.file.commit": {
          const payload = activity.payload as ActivityTaskFileCommitPayload;
          return `committed ${payload.filePath} to ${payload.branch}@${payload.repositoryFullPath}`;
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
};
</script>
