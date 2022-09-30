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
          <li v-for="(activity, index) in activityList" :key="activity.id">
            <div :id="`#activity${activity.id}`" class="relative pb-4">
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
                        <ActivityActionSentence
                          :issue="issue"
                          :activity="activity"
                        />

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
                      v-if="allowEditActivity(activity)"
                      class="space-x-2 flex items-center text-control-light"
                    >
                      <template
                        v-if="
                          state.editCommentMode &&
                          state.activeActivity?.id === activity.id
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
                        state.activeActivity?.id === activity.id
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
                          (e: any) => {
                            sizeToFit(e.target);
                          }
                        "
                        @focus="
                          (e: any) => {
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
                        :href="fileCommitActivityUrl(activity)"
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
                  (e: any) => {
                    sizeToFit(e.target);
                  }
                "
              ></textarea>
              <div class="mt-4 flex items-center justify-between space-x-4">
                <div>
                  <button
                    type="button"
                    class="group btn-normal !text-accent hover:!bg-gray-50"
                    @click="lgtm"
                  >
                    <heroicons-outline:thumb-up
                      class="w-5 h-5 group-hover:thumb-up"
                    />
                    <span class="ml-1">LGTM</span>
                  </button>
                </div>
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
  <BBAlert
    v-if="state.showDeleteCommentModal"
    :style="'INFO'"
    :ok-text="'Delete'"
    :title="'Are you sure to delete this comment?'"
    :description="'You cannot undo this action.'"
    @ok="
      () => {
        doDeleteComment(state.activeActivity!);
        state.showDeleteCommentModal = false;
        state.activeActivity = undefined;
      }
    "
    @cancel="state.showDeleteCommentModal = false"
  >
  </BBAlert>
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
import PrincipalAvatar from "../PrincipalAvatar.vue";
import type {
  Issue,
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ActivityCreate,
  IssueSubscriber,
  ActivityTaskFileCommitPayload,
  Task,
} from "@/types";
import { UNKNOWN_ID, EMPTY_ID, SYSTEM_BOT_ID } from "@/types";
import { findTaskById, issueSlug, sizeToFit, taskSlug } from "@/utils";
import { IssueBuiltinFieldId } from "@/plugins";
import { useI18n } from "vue-i18n";
import {
  useCurrentUser,
  useUIStateStore,
  useIssueSubscriberStore,
  useActivityStore,
} from "@/store";
import { useEventListener } from "@vueuse/core";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import ActivityActionSentence from "./activity/ActionSentence.vue";

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

const emit = defineEmits<{
  (event: "run-checks", task: Task): void;
}>();

const { t } = useI18n();
const activityStore = useActivityStore();
const route = useRoute();

const newComment = ref("");
const newCommentTextArea = ref();
const editComment = ref("");
const editCommentTextArea = ref();

const logic = useIssueLogic();
const issue = logic.issue as Ref<Issue>;
const { addSubscriberId } = useExtraIssueLogic();

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
      doCreateComment(newComment.value);
    }
  }
};

useEventListener("keydown", keyboardHandler);

const currentUser = useCurrentUser();

const prepareActivityList = () => {
  activityStore.fetchActivityListForIssue(issue.value);
};

watchEffect(prepareActivityList);

// Need to use computed to make list reactive to activity list changes.
const activityList = computed((): Activity[] => {
  const list = activityStore.getActivityListByIssue(issue.value.id);
  return list.filter((activity: Activity) => {
    if (activity.type == "bb.issue.field.update") {
      const containUserVisibleChange =
        (activity.payload as ActivityIssueFieldUpdatePayload).fieldId !=
        IssueBuiltinFieldId.SUBSCRIBER_LIST;
      return containUserVisibleChange;
    }
    return true;
  });
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
    useUIStateStore().saveIntroStateByKey({
      key: "comment.create",
      newState: true,
    });

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

    if (comment === "LGTM") {
      emit("run-checks", logic.selectedTask.value as Task);
    }
  });
};

const lgtm = (e: Event) => {
  doCreateComment("LGTM", false);

  // import the effect lib asynchronously
  import("canvas-confetti").then(({ default: confetti }) => {
    const button = e.target as HTMLElement;
    const { left, top, width, height } = button.getBoundingClientRect();
    const { innerWidth: winWidth, innerHeight: winHeight } = window;
    // Create a confetti effect from the position of the LGTM button
    confetti({
      particleCount: 100,
      spread: 70,
      origin: {
        x: (left + width / 2) / winWidth,
        y: (top + height / 2) / winHeight,
      },
    });
  });
};

const allowEditActivity = (activity: Activity) => {
  return (
    activity.type === "bb.issue.comment.create" &&
    currentUser.value.id === activity.creator.id
  );
};

const onUpdateComment = (activity: Activity) => {
  editComment.value = activity.comment;
  state.activeActivity = activity;
  state.editCommentMode = true;
  nextTick(() => {
    editCommentTextArea.value?.focus();
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

const doDeleteComment = (activity: Activity) => {
  activityStore.deleteActivity(activity);
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
      case "PENDING_APPROVAL": {
        return "avatar"; // stale approval dismissed.
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
      if (issue.value.pipeline.id != EMPTY_ID) {
        const payload = activity.payload as ActivityTaskStatusUpdatePayload;
        const task = findTaskById(issue.value.pipeline, payload.taskId);
        let link = "";
        if (task.id != UNKNOWN_ID) {
          link = `/issue/${issueSlug(
            issue.value.name,
            issue.value.id
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

const fileCommitActivityUrl = (activity: Activity) => {
  const payload = activity.payload as ActivityTaskFileCommitPayload;
  return `${payload.vcsInstanceUrl}/${payload.repositoryFullPath}/-/commit/${payload.commitId}`;
};

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
