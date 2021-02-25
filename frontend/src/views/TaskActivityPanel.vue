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
              <div :id="'activity' + (index + 1)" class="relative pb-8">
                <span
                  v-if="index != activityList.length - 1"
                  class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
                  aria-hidden="true"
                ></span>
                <div class="relative flex items-start space-x-3">
                  <template
                    v-if="
                      activity.attributes.actionType == 'bytebase.task.create'
                    "
                  >
                    <div>
                      <div class="relative pl-0.5 pt-0.5">
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
                  <div class="min-w-0 flex-1">
                    <div class="min-w-0 flex-1 py-1.5">
                      <div class="text-sm text-control-light">
                        <span class="font-medium text-main">{{
                          activity.attributes.creator.name
                        }}</span>
                        {{ actionSentence(activity.attributes.actionType) }}
                        <a
                          :href="'#activity' + (index + 1)"
                          class="link whitespace-nowrap"
                          >{{ humanizeTs(activity.attributes.createdTs) }}</a
                        >
                      </div>
                    </div>
                    <template
                      v-if="
                        activity.attributes.actionType ==
                        'bytebase.task.comment.create'
                      "
                    >
                      <div class="mt-2 text-sm text-control">
                        {{ activity.attributes.payload.content }}
                      </div>
                    </template>
                  </div>
                </div>
              </div>
            </li>
          </ul>
        </div>

        <div class="mt-6">
          <div class="flex space-x-3">
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
            <div class="min-w-0 flex-1">
              <BBAutoResize>
                <template v-slot:default="{ resize }">
                  <label for="comment" class="sr-only">Comment</label>
                  <textarea
                    id="comment"
                    name="comment"
                    rows="3"
                    class="resize-none shadow-sm block w-full focus:ring-gray-900 focus:border-gray-900 sm:text-sm border-gray-300 rounded-md"
                    placeholder="Leave a comment..."
                    v-model="comment"
                    @input="
                      (e) => {
                        resize(e);
                      }
                    "
                  ></textarea>
                </template>
              </BBAutoResize>
              <div class="mt-2 flex items-center justify-start space-x-4">
                <button
                  type="button"
                  class="btn-normal"
                  ref="commentButton"
                  :disabled="comment.length == 0"
                  @click.prevent="createComment"
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
</template>

<script lang="ts">
import { computed, inject, ref, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { User, Task, TaskActionType } from "../types";

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
    const comment = ref("");
    const commentButton = ref();

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

    const createComment = () => {
      console.log(comment.value);
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
      comment,
      commentButton,
      currentUser,
      activityList,
      actionSentence,
      createComment,
    };
  },
};
</script>
