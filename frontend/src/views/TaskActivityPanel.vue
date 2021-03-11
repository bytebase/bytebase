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
            <div :id="'activity' + (index + 1)" class="relative pb-6">
              <span
                v-if="index != activityList.length - 1"
                class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
                aria-hidden="true"
              ></span>
              <div class="relative flex items-start">
                <template v-if="activity.actionType == 'bytebase.task.create'">
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-8 ring-white flex items-center justify-center"
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
                <template
                  v-else-if="
                    activity.actionType == 'bytebase.task.field.update'
                  "
                >
                  <div class="relative pl-0.5">
                    <div
                      class="w-7 h-7 bg-control-bg rounded-full ring-8 ring-white flex items-center justify-center"
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
                <template v-else>
                  <div class="relative">
                    <BBAvatar
                      class="rounded-full ring-8 ring-white"
                      :username="activity.creator.name"
                    >
                    </BBAvatar>
                  </div>
                </template>
                <div class="ml-3 min-w-0 flex-1">
                  <div class="min-w-0 flex-1 pt-1 flex justify-between">
                    <div class="text-sm text-control-light">
                      <span class="font-medium text-main whitespace-nowrap">{{
                        activity.creator.name
                      }}</span>
                      <a
                        :href="'#activity' + (index + 1)"
                        class="ml-1 anchor-link whitespace-normal"
                      >
                        {{ actionSentence(activity) }}
                        {{ humanizeTs(activity.createdTs) }}
                        <template
                          v-if="
                            activity.createdTs != activity.lastUpdatedTs &&
                            activity.actionType ==
                              'bytebase.task.comment.create'
                          "
                        >
                          (edited
                          {{ humanizeTs(activity.createdTs) }})
                        </template>
                      </a>
                    </div>
                    <div
                      v-if="
                        currentUser.id == activity.creator.id &&
                        activity.actionType == 'bytebase.task.comment.create'
                      "
                      class="space-x-2 flex items-center text-control-light"
                    >
                      <template
                        v-if="
                          state.editCommentMode &&
                          state.activeComment.id == activity.id
                        "
                      >
                        <button
                          type="button"
                          class="rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                          @click.prevent="cancelEditComment"
                        >
                          Cancel
                        </button>
                        <button
                          type="button"
                          class="border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                          :disabled="
                            editComment.length == 0 ||
                            editComment == activity.payload.comment
                          "
                          @click.prevent="doUpdateComment"
                        >
                          Save
                        </button>
                      </template>
                      <!-- mr-2 is to veritical align with the text description edit button-->
                      <div v-else class="mr-2 flex items-center space-x-2">
                        <!-- Delete Comment Button-->
                        <button
                          class="btn-icon"
                          @click.prevent="
                            {
                              state.activeComment = activity;
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
                  <template
                    v-if="activity.actionType == 'bytebase.task.comment.create'"
                  >
                    <div class="mt-2 text-sm text-control whitespace-pre-wrap">
                      <template
                        v-if="
                          state.editCommentMode &&
                          state.activeComment.id == activity.id
                        "
                      >
                        <label for="comment" class="sr-only"
                          >Edit Comment</label
                        >
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
                        {{ activity.payload.comment }}
                      </template>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </li>
        </ul>

        <div v-if="!state.editCommentMode">
          <div class="flex">
            <div class="flex-shrink-0">
              <div class="relative">
                <BBAvatar :username="currentUser.name"> </BBAvatar>
                <span
                  class="absolute -bottom-0.5 -right-1 bg-white rounded-tl px-0.5 py-px"
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
        doDeleteComment(state.activeComment);
        state.showDeleteCommentModal = false;
        state.activeComment = null;
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
import {
  Task,
  Activity,
  ActionTaskCommentCreatePayload,
  ActionTaskFieldUpdatePayload,
  Environment,
} from "../types";
import { sizeToFit } from "../utils";
import { fieldFromId, TaskTemplate, TaskBuiltinFieldId } from "../plugins";

interface LocalState {
  showDeleteCommentModal: boolean;
  editCommentMode: boolean;
  activeComment?: Activity;
}

export default {
  name: "TaskActivityPanel",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    taskTemplate: {
      required: true,
      type: Object as PropType<TaskTemplate>,
    },
  },
  components: {},
  setup(props, ctx) {
    const store = useStore();
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
          doUpdateComment();
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
      store
        .dispatch("activity/fetchActivityListForTask", props.task.id)
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareActivityList);

    const activityList = computed(() =>
      store.getters["activity/activityListByTask"](props.task.id)
    );

    const cancelEditComment = () => {
      editComment.value = "";
      state.activeComment = undefined;
      state.editCommentMode = false;
    };

    const doCreateComment = () => {
      store
        .dispatch("activity/createActivity", {
          actionType: "bytebase.task.comment.create",
          containerId: props.task.id,
          creator: {
            id: currentUser.value.id,
            name: currentUser.value.name,
          },
          payload: {
            comment: newComment.value,
          },
        })
        .then(() => {
          newComment.value = "";
          nextTick(() => sizeToFit(newCommentTextArea.value));
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const onUpdateComment = (activity: Activity) => {
      editComment.value = (activity.payload! as ActionTaskCommentCreatePayload).comment;
      state.activeComment = activity;
      state.editCommentMode = true;
      nextTick(() => {
        editCommentTextArea.value.focus();
      });
    };

    const doUpdateComment = () => {
      const activityPatch = store
        .dispatch("activity/updateComment", {
          activityId: state.activeComment!.id,
          updatedComment: editComment.value,
        })
        .then(() => {
          cancelEditComment();
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDeleteComment = (activity: Activity) => {
      store.dispatch("activity/deleteActivity", activity).catch((error) => {
        console.log(error);
      });
    };

    const actionSentence = (activity: Activity): string => {
      switch (activity.actionType) {
        case "bytebase.task.create":
          return "created task";
        case "bytebase.task.comment.create":
          return "commented";
        case "bytebase.task.field.update": {
          const updateInfoList: string[] = [];
          for (const update of (activity.payload as ActionTaskFieldUpdatePayload)
            ?.changeList || []) {
            let name = "Unknown Field";
            let oldValue = undefined;
            let newValue = undefined;
            if (update.fieldId == TaskBuiltinFieldId.STATUS) {
              name = "Status";
              oldValue = update.oldValue;
              newValue = update.newValue;
            } else if (update.fieldId == TaskBuiltinFieldId.ASSIGNEE) {
              name = "Assignee";
              if (update.oldValue) {
                oldValue = store.getters["principal/principalById"](
                  update.oldValue
                ).name;
              }
              if (update.newValue) {
                newValue = store.getters["principal/principalById"](
                  update.newValue
                ).name;
              }
            } else if (update.fieldId == TaskBuiltinFieldId.DESCRIPTION) {
              name = "Description";
            } else {
              const field = fieldFromId(props.taskTemplate, update.fieldId);
              if (field) {
                name = field.name;
                if (field.type === "String") {
                  oldValue = update.oldValue;
                  newValue = update.newValue;
                } else if (field.type === "Environment") {
                  if (update.oldValue) {
                    const environment: Environment = store.getters[
                      "environment/environmentById"
                    ](update.oldValue);
                    if (environment) {
                      oldValue = environment.name;
                    } else {
                      oldValue = "Unknown Environment";
                    }
                  }
                  if (update.newValue) {
                    const environment: Environment = store.getters[
                      "environment/environmentById"
                    ](update.newValue);
                    if (environment) {
                      newValue = environment.name;
                    } else {
                      newValue = "Unknown Environment";
                    }
                  }
                }
              }
            }
            if (oldValue && newValue) {
              updateInfoList.push(
                "changed " +
                  name +
                  ' from "' +
                  oldValue +
                  '" to "' +
                  newValue +
                  '"'
              );
            } else if (oldValue) {
              updateInfoList.push("unset " + name + ' from "' + oldValue + '"');
            } else if (newValue) {
              updateInfoList.push("set " + name + ' to "' + newValue + '"');
            } else {
              updateInfoList.push("changed " + name);
            }
          }
          if (updateInfoList.length > 0) {
            return updateInfoList.join("; ");
          }
          return "updated";
        }
      }
    };

    return {
      state,
      newComment,
      newCommentTextArea,
      editComment,
      editCommentTextArea,
      currentUser,
      activityList,
      actionSentence,
      doCreateComment,
      cancelEditComment,
      onUpdateComment,
      doUpdateComment,
      doDeleteComment,
    };
  },
};
</script>
