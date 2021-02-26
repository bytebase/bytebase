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
        <div class="flow-root">
          <ul class="-mb-8">
            <li v-for="(activity, index) in activityList" :key="index">
              <div :id="'activity' + (index + 1)" class="relative pb-6">
                <span
                  v-if="index != activityList.length - 1"
                  class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
                  aria-hidden="true"
                ></span>
                <div class="relative flex items-start">
                  <template
                    v-if="
                      activity.attributes.actionType == 'bytebase.task.create'
                    "
                  >
                    <div>
                      <div class="relative pl-0.5">
                        <div
                          class="bg-control-bg rounded-full ring-8 ring-white flex items-center justify-center"
                        >
                          <svg
                            class="w-7 h-7 text-control"
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
                    </div>
                  </template>
                  <template
                    v-else-if="
                      activity.attributes.actionType ==
                      'bytebase.task.comment.create'
                    "
                  >
                    <div class="relative">
                      <BBAvatar
                        class="rounded-full ring-8 ring-white"
                        :username="currentUser.attributes.name"
                      >
                      </BBAvatar>
                    </div>
                  </template>
                  <div class="ml-3 min-w-0 flex-1">
                    <div class="min-w-0 flex-1 pt-1 flex justify-between">
                      <div class="text-sm text-control-light">
                        <span class="font-medium text-main">{{
                          activity.attributes.creator.name
                        }}</span>
                        <a
                          :href="'#activity' + (index + 1)"
                          class="ml-1 anchor-link whitespace-nowrap"
                        >
                          {{ actionSentence(activity.attributes.actionType) }}
                          {{ humanizeTs(activity.attributes.createdTs) }}
                          <template
                            v-if="
                              activity.attributes.createdTs !=
                                activity.attributes.lastUpdatedTs &&
                              activity.attributes.actionType ==
                                'bytebase.task.comment.create'
                            "
                          >
                            (edited
                            {{ humanizeTs(activity.attributes.createdTs) }})
                          </template>
                        </a>
                      </div>
                      <div
                        v-if="currentUser.id == activity.attributes.creator.id"
                        class="space-x-2 text-control-light"
                      >
                        <template
                          v-if="
                            state.editCommentMode &&
                            state.activeComment.id == activity.id
                          "
                        >
                          <button
                            type="button"
                            class="text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                            @click.prevent="
                              {
                                editComment = '';
                                state.activeComment = null;
                                state.editCommentMode = false;
                              }
                            "
                          >
                            Cancel
                          </button>
                          <button
                            type="button"
                            class="border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed px-2 text-xs leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
                            :disabled="
                              editComment.length == 0 ||
                              editComment == activity.attributes.payload.content
                            "
                            @click.prevent="doUpdateComment"
                          >
                            Save
                          </button>
                        </template>
                        <template v-else>
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
                        </template>
                      </div>
                    </div>
                    <template
                      v-if="
                        activity.attributes.actionType ==
                        'bytebase.task.comment.create'
                      "
                    >
                      <div
                        class="mt-2 text-sm text-control whitespace-pre-wrap"
                      >
                        <template
                          v-if="
                            state.editCommentMode &&
                            state.activeComment.id == activity.id
                          "
                        >
                          <BBAutoResize>
                            <template v-slot:main="{ resize }">
                              <label for="comment" class="sr-only"
                                >Edit Comment</label
                              >
                              <textarea
                                ref="editCommentTextArea"
                                rows="3"
                                class="resize-none shadow-sm block w-full focus:ring-gray-900 focus:border-gray-900 sm:text-sm border-gray-300 rounded-md"
                                placeholder="Leave a comment..."
                                v-model="editComment"
                                @input="
                                  (e) => {
                                    resize(e.target);
                                  }
                                "
                                @focus="
                                  (e) => {
                                    resize(e.target);
                                  }
                                "
                              ></textarea>
                            </template>
                          </BBAutoResize>
                        </template>
                        <template v-else>
                          {{ activity.attributes.payload.content }}
                        </template>
                      </div>
                    </template>
                  </div>
                </div>
              </div>
            </li>
          </ul>
        </div>

        <div v-if="!state.editCommentMode" class="mt-8">
          <div class="flex">
            <div class="flex-shrink-0">
              <div class="relative">
                <BBAvatar :username="currentUser.attributes.name"> </BBAvatar>
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
              <BBAutoResize>
                <template v-slot:main="{ resize }">
                  <label for="comment" class="sr-only">Create Comment</label>
                  <textarea
                    ref="newCommentTextArea"
                    rows="3"
                    class="resize-none shadow-sm block w-full focus:ring-gray-900 focus:border-gray-900 sm:text-sm border-gray-300 rounded-md"
                    placeholder="Leave a comment..."
                    v-model="newComment"
                    @input="
                      (e) => {
                        resize(e.target);
                      }
                    "
                  ></textarea>
                </template>
                <template v-slot:accessory="{ resize }">
                  <div class="mt-4 flex items-center justify-start space-x-4">
                    <button
                      type="button"
                      class="btn-normal"
                      :disabled="newComment.length == 0"
                      @click.prevent="doCreateComment(resize)"
                    >
                      Comment
                    </button>
                  </div>
                </template>
              </BBAutoResize>
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
  computed,
  inject,
  nextTick,
  ref,
  reactive,
  watchEffect,
  PropType,
} from "vue";
import { useStore } from "vuex";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { User, Task, TaskActionType, Activity, ActivityId } from "../types";

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

    const currentUser = inject<User>(UserStateSymbol);

    const prepareActivityList = () => {
      store
        .dispatch("activity/fetchActivityListForTask", props.task.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const activityList = computed(() =>
      store.getters["activity/activityListByTask"](props.task.id)
    );

    const doCreateComment = (resize: (el: HTMLTextAreaElement) => void) => {
      store
        .dispatch("activity/createActivity", {
          type: "activity",
          attributes: {
            actionType: "bytebase.task.comment.create",
            containerId: props.task.id,
            creator: {
              id: currentUser!.id,
              name: currentUser!.attributes.name,
            },
            payload: {
              content: newComment.value,
            },
          },
        })
        .then(() => {
          newComment.value = "";
          nextTick(() => resize(newCommentTextArea.value));
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const onUpdateComment = (activity: Activity) => {
      editComment.value = activity.attributes.payload!.content;
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
          editComment.value = "";
          state.activeComment = undefined;
          state.editCommentMode = false;
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

    watchEffect(prepareActivityList);

    const actionSentence = (actionType: TaskActionType) => {
      switch (actionType) {
        case "bytebase.task.create":
          return "created task";
        case "bytebase.task.comment.create":
          return "commented";
        case "bytebase.task.field.update":
          return "updated";
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
      onUpdateComment,
      doUpdateComment,
      doDeleteComment,
    };
  },
};
</script>
