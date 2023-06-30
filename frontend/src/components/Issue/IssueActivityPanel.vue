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
            :key="item.activity.name"
            :activity-list="activityList"
            :issue="issue"
            :index="index"
            :activity="item.activity"
            :similar="item.similar"
          >
            <template v-if="allowEditActivity(item.activity)" #subject-suffix>
              <div class="space-x-2 flex items-center text-control-light">
                <!-- mr-2 is to vertical align with the text description edit button-->
                <div
                  v-if="!state.editCommentMode"
                  class="mr-2 flex items-center space-x-2"
                >
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
              <template v-if="actuatorStore.version !== 'development'">
                <div
                  v-if="
                    state.editCommentMode &&
                    state.activeActivity?.name === item.activity.name
                  "
                  class="mt-2 text-sm text-control whitespace-pre-wrap"
                >
                  <label for="comment" class="sr-only">
                    {{ $t("issue.edit-comment") }}
                  </label>
                  <textarea
                    ref="editCommentTextArea"
                    v-model="state.editComment"
                    rows="3"
                    class="textarea block w-full resize-none"
                    :placeholder="$t('issue.leave-a-comment')"
                    @input="(e: any) => sizeToFit(e.target)"
                    @focus="(e: any) => sizeToFit(e.target)"
                  />
                </div>
                <ActivityComment v-else :activity="item.activity" />
              </template>
              <MarkdownEditor
                v-else-if="item.activity.comment"
                :mode="
                  state.editCommentMode &&
                  state.activeActivity?.name === item.activity.name
                    ? 'editor'
                    : 'preview'
                "
                :content="item.activity.comment"
                @change="(val: string) => state.editComment = val"
                @submit="doUpdateComment"
                @cancel="cancelEditComment"
              />
              <div
                v-if="
                  state.editCommentMode &&
                  state.activeActivity?.name === item.activity.name
                "
                class="flex space-x-2 mt-4 items-center justify-end"
              >
                <button
                  type="button"
                  class="btn-normal border-none"
                  @click.prevent="cancelEditComment"
                >
                  {{ $t("common.cancel") }}
                </button>
                <button
                  type="button"
                  class="btn-normal"
                  :disabled="!allowUpdateComment"
                  @click.prevent="doUpdateComment"
                >
                  {{ $t("common.save") }}
                </button>
              </div>
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
              <MarkdownEditor
                v-if="actuatorStore.version === 'development'"
                mode="editor"
                :content="state.newComment"
                @change="(val: string) => state.newComment = val"
                @submit="doCreateComment(state.newComment)"
              />
              <textarea
                v-else
                ref="newCommentTextArea"
                v-model="state.newComment"
                rows="3"
                class="textarea block w-full resize-none whitespace-pre-wrap"
                :placeholder="$t('issue.leave-a-comment')"
                @input="(e: any) => sizeToFit(e.target)"
              ></textarea>
              <div class="mt-4 flex items-center justify-between space-x-4">
                <div>
                  <button
                    type="button"
                    class="btn-normal"
                    :disabled="state.newComment.length == 0"
                    @click.prevent="doCreateComment(state.newComment)"
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
  ActivityIssueFieldUpdatePayload,
  IssueSubscriber,
  ActivityIssueCommentCreatePayload,
} from "@/types";
import { extractUserResourceName, sizeToFit } from "@/utils";
import { IssueBuiltinFieldId } from "@/plugins";
import {
  useIssueSubscriberStore,
  useReviewV1Store,
  useActivityV1Store,
  useCurrentUserV1,
  useCurrentUser,
  useActuatorV1Store,
} from "@/store";
import { useEventListener } from "@vueuse/core";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import {
  ActivityItem,
  DistinctActivity,
  Comment as ActivityComment,
  isSimilarActivity,
} from "./activity";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import { getLogId } from "@/store/modules/v1/common";

interface LocalState {
  editCommentMode: boolean;
  activeActivity?: LogEntity;
  editComment: string;
  newComment: string;
}

const reviewV1Store = useReviewV1Store();
const activityV1Store = useActivityV1Store();
const actuatorStore = useActuatorV1Store();
const route = useRoute();

const newCommentTextArea = ref();
const editCommentTextArea = ref<HTMLTextAreaElement[]>();

const logic = useIssueLogic();
const issue = logic.issue as Ref<Issue>;
const { addSubscriberId } = useExtraIssueLogic();

const state = reactive<LocalState>({
  editCommentMode: false,
  editComment: "",
  newComment: "",
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
      doCreateComment(state.newComment);
    }
  }
};

useEventListener("keydown", keyboardHandler);

const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();

const prepareActivityList = () => {
  activityV1Store.fetchActivityListForIssue(issue.value);
};

watchEffect(prepareActivityList);

// Need to use computed to make list reactive to activity list changes.
const activityList = computed((): DistinctActivity[] => {
  const list = activityV1Store
    .getActivityListByIssue(issue.value.id)
    .filter((activity) => {
      if (activity.action === LogEntity_Action.ACTION_ISSUE_APPROVAL_NOTIFY) {
        return false;
      }

      if (activity.action === LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE) {
        const containUserVisibleChange =
          (JSON.parse(activity.payload) as ActivityIssueFieldUpdatePayload)
            .fieldId !== IssueBuiltinFieldId.SUBSCRIBER_LIST;
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
  state.activeActivity = undefined;
  state.editCommentMode = false;
  state.editComment = "";
};

const doCreateComment = (comment: string) => {
  reviewV1Store
    .createReviewComment({
      reviewId: issue.value.id,
      comment,
    })
    .then(() => {
      state.newComment = "";
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
        addSubscriberId(currentUser.value.id);
      }
    });
};

const allowEditActivity = (activity: LogEntity) => {
  if (activity.action !== LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE) {
    return false;
  }
  if (currentUserV1.value.email !== extractUserResourceName(activity.creator)) {
    return false;
  }
  const payload = JSON.parse(
    activity.payload
  ) as ActivityIssueCommentCreatePayload;
  if (payload && payload.externalApprovalEvent) {
    return false;
  }
  return true;
};

const onUpdateComment = (activity: LogEntity) => {
  state.activeActivity = activity;
  state.editCommentMode = true;
  state.editComment = activity.comment;
  nextTick(() => {
    editCommentTextArea.value?.[0]?.focus();
  });
};

const doUpdateComment = () => {
  if (!state.activeActivity) {
    return;
  }
  if (!state.editComment) {
    return;
  }
  const activityId = getLogId(state.activeActivity.name);
  reviewV1Store
    .updateReviewComment({
      commentId: `${activityId}`,
      reviewId: issue.value.id,
      comment: state.editComment,
    })
    .then(() => {
      cancelEditComment();
    });
};

const allowUpdateComment = computed(() => {
  return (
    state.editComment && state.editComment != state.activeActivity!.comment
  );
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
