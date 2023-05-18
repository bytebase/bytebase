<template>
  <div id="activity">
    <div class="divide-y divide-block-border">
      <div class="pb-4">
        <h2 id="activity-title" class="text-lg font-medium text-main">
          {{ $t("common.activity") }}
        </h2>
      </div>
      <div class="pt-6">
        <!-- Activity feed-->
        <ul>
          <ActivityItem
            v-for="(item, index) in activityList"
            :key="item.activity.id"
            :activity-list="activityList"
            :issue="issue"
            :index="index"
            :activity="item.activity"
            :similar="item.similar"
          >
            <template v-if="allowEditActivity(item.activity)" #subject-suffix>
              <div class="space-x-2 flex items-center text-control-light">
                <template
                  v-if="
                    state.editCommentMode &&
                    state.activeActivity?.id === item.activity.id
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
                  <!-- Edit Comment Button-->
                  <button
                    class="btn-icon"
                    @click.prevent="onUpdateComment(item.activity)"
                  >
                    <heroicons-outline:pencil class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </template>

            <template #comment>
              <div
                v-if="
                  state.editCommentMode &&
                  state.activeActivity?.id === item.activity.id
                "
                class="mt-2 text-sm text-control whitespace-pre-wrap"
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
                  (e: any) => {
                    sizeToFit(e.target);
                  }
                "
                  @focus="
                  (e: any) => {
                    sizeToFit(e.target);
                  }
                "
                />
              </div>
              <ActivityComment v-else :activity="item.activity" />
            </template>
          </ActivityItem>
        </ul>

        <div v-if="!state.editCommentMode">
          <div class="flex">
            <div class="flex-shrink-0">
              <div class="relative">
                <UserAvatar :user="currentUserV1" />
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
                  (e: any) => {
                    sizeToFit(e.target);
                  }
                "
              ></textarea>
              <div class="mt-4 flex items-center justify-between space-x-4">
                <div>
                  <button
                    type="button"
                    class="btn-normal"
                    :disabled="newComment.length == 0"
                    @click.prevent="doCreateComment(newComment)"
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
  </div>
</template>

<script lang="ts" setup>
import {
  computed,
  nextTick,
  ref,
  reactive,
  watch,
  watchEffect,
  Ref,
  onMounted,
} from "vue";
import { useRoute } from "vue-router";
import UserAvatar from "../User/UserAvatar.vue";
import type {
  Issue,
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityCreate,
  IssueSubscriber,
  ActivityIssueCommentCreatePayload,
} from "@/types";
import { extractUserUID, sizeToFit } from "@/utils";
import { IssueBuiltinFieldId } from "@/plugins";
import {
  useIssueSubscriberStore,
  useActivityStore,
  useCurrentUserV1,
  useCurrentUser,
} from "@/store";
import { useEventListener } from "@vueuse/core";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import {
  ActivityItem,
  DistinctActivity,
  Comment as ActivityComment,
  isSimilarActivity,
} from "./activity";

interface LocalState {
  editCommentMode: boolean;
  activeActivity?: Activity;
}

const activityStore = useActivityStore();
const route = useRoute();

const newComment = ref("");
const newCommentTextArea = ref();
const editComment = ref("");
const editCommentTextArea = ref<HTMLTextAreaElement[]>();

const logic = useIssueLogic();
const issue = logic.issue as Ref<Issue>;
const { addSubscriberId } = useExtraIssueLogic();

const state = reactive<LocalState>({
  editCommentMode: false,
});

const keyboardHandler = (e: KeyboardEvent) => {
  if (
    state.editCommentMode &&
    editCommentTextArea.value?.[0] === document.activeElement
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
      doCreateComment(newComment.value);
    }
  }
};

useEventListener("keydown", keyboardHandler);

const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();

const prepareActivityList = () => {
  activityStore.fetchActivityListForIssue(issue.value);
};

watchEffect(prepareActivityList);

// Need to use computed to make list reactive to activity list changes.
const activityList = computed((): DistinctActivity[] => {
  const list = activityStore
    .getActivityListByIssue(issue.value.id)
    .filter((activity: Activity) => {
      if (activity.type === "bb.issue.field.update") {
        const containUserVisibleChange =
          (activity.payload as ActivityIssueFieldUpdatePayload).fieldId !==
          IssueBuiltinFieldId.SUBSCRIBER_LIST;
        return containUserVisibleChange;
      }
      return true;
    });

  const distinctActivityList: DistinctActivity[] = [];
  for (let i = 0; i < list.length; i++) {
    const activity = list[i];
    if (distinctActivityList.length === 0) {
      distinctActivityList.push({ activity, similar: [] });
      continue;
    }

    const prev = distinctActivityList[distinctActivityList.length - 1];
    if (isSimilarActivity(prev.activity, activity)) {
      prev.similar.push(activity);
    } else {
      distinctActivityList.push({ activity, similar: [] });
    }
  }

  return distinctActivityList;
});

const subscriberList = computed((): IssueSubscriber[] => {
  return useIssueSubscriberStore().subscriberListByIssue(issue.value.id);
});

const cancelEditComment = () => {
  editComment.value = "";
  state.activeActivity = undefined;
  state.editCommentMode = false;
};

const doCreateComment = (comment: string, clear = true) => {
  const createActivity: ActivityCreate = {
    type: "bb.issue.comment.create",
    containerId: issue.value.id,
    comment,
  };
  activityStore.createActivity(createActivity).then(() => {
    if (clear) {
      newComment.value = "";
      nextTick(() => sizeToFit(newCommentTextArea.value));
    }

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
      addSubscriberId(currentUser.value.id);
    }
  });
};

const allowEditActivity = (activity: Activity) => {
  if (activity.type !== "bb.issue.comment.create") {
    return false;
  }
  if (
    extractUserUID(currentUserV1.value.name) !== String(activity.creator.id)
  ) {
    return false;
  }
  const payload = activity.payload as ActivityIssueCommentCreatePayload;
  if (payload && payload.externalApprovalEvent) {
    return false;
  }
  return true;
};

const onUpdateComment = (activity: Activity) => {
  editComment.value = activity.comment;
  state.activeActivity = activity;
  state.editCommentMode = true;
  nextTick(() => {
    editCommentTextArea.value?.[0]?.focus();
  });
};

const doUpdateComment = () => {
  activityStore
    .updateComment({
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

onMounted(() => {
  watch(
    () => route.hash,
    (hash) => {
      if (hash.match(/^#activity(\d+)/)) {
        // use '#activity' element as a fallback
        const elem =
          document.querySelector(hash) || document.querySelector("#activity");
        // We use `setTimeout` here since this should be executed very late.
        setTimeout(() => elem?.scrollIntoView());
      }
    },
    {
      immediate: true,
    }
  );
});
</script>
