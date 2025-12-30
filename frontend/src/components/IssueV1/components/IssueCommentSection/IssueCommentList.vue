<template>
  <div class="flex flex-col">
    <ul>
      <IssueCreatedCommentV1
        :issue-comments="issueComments"
        :issue="issue"
        @update-issue="handleUpdateIssue"
      />
      <IssueCommentView
        class="group"
        v-for="(item, index) in issueComments"
        :key="item.name"
        :issue="issue"
        :index="index"
        :is-last="index === issueComments.length - 1"
        :issue-comment="item"
      >
        <template v-if="allowEditIssueComment(item)" #subject-suffix>
          <!-- Edit Comment Button-->
          <NButton
            v-if="!state.editCommentMode"
            quaternary
            size="tiny"
            class="text-gray-500 hover:text-gray-700"
            @click.prevent="onUpdateComment(item)"
          >
            <PencilIcon class="w-3.5 h-3.5" />
          </NButton>
        </template>

        <template v-if="item.comment" #comment>
          <MarkdownEditor
            :mode="
              state.editCommentMode &&
              state.activeComment?.name === item.name
                ? 'editor'
                : 'preview'
            "
            :content="item.comment"
            :project="project"
            :maxlength="65536"
            :max-height="Number.MAX_SAFE_INTEGER"
            @change="(val: string) => (state.editComment = val)"
            @submit="doUpdateComment"
          />
          <div
            v-if="
              state.editCommentMode &&
              state.activeComment?.name === item.name
            "
            class="flex gap-x-2 mt-2 items-center justify-end"
          >
            <NButton quaternary size="small" @click.prevent="cancelEditComment">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              size="small"
              :disabled="!allowUpdateComment"
              @click.prevent="doUpdateComment"
            >
              {{ $t("common.save") }}
            </NButton>
          </div>
        </template>
      </IssueCommentView>
    </ul>

    <div v-if="!state.editCommentMode && allowCreateComment" class="mt-2">
      <div class="flex gap-3">
        <div class="shrink-0 pt-1">
          <UserAvatar :user="currentUser" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="sr-only" id="issue-comment-editor"></h3>
          <MarkdownEditor
            mode="editor"
            :content="state.newComment"
            :project="project"
            :maxlength="65536"
            @change="(val: string) => (state.newComment = val)"
            @submit="doCreateComment(state.newComment)"
          />
          <div class="mt-3 flex items-center justify-end">
            <NButton
              type="primary"
              :disabled="state.newComment.length == 0"
              @click.prevent="doCreateComment(state.newComment)"
            >
              {{ $t("common.comment") }}
            </NButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { PencilIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, watch, watchEffect } from "vue";
import { useRoute } from "vue-router";
import MarkdownEditor from "@/components/MarkdownEditor";
import UserAvatar from "@/components/User/UserAvatar.vue";
import {
  extractUserId,
  getIssueCommentType,
  IssueCommentType,
  useCurrentProjectV1,
  useCurrentUserV1,
  useIssueCommentStore,
} from "@/store";
import type { ComposedIssue } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useIssueContext } from "../../logic";
import { IssueCommentView } from "./IssueCommentView";
import IssueCreatedCommentV1 from "./IssueCommentView/IssueCreatedCommentV1.vue";

interface LocalState {
  editCommentMode: boolean;
  activeComment?: IssueComment;
  editComment: string;
  newComment: string;
}

const route = useRoute();

const { project } = useCurrentProjectV1();
const { issue } = useIssueContext();

const state = reactive<LocalState>({
  editCommentMode: false,
  editComment: "",
  newComment: "",
});

const currentUser = useCurrentUserV1();
const issueCommentStore = useIssueCommentStore();

const prepareIssueComments = async () => {
  await issueCommentStore.listIssueComments(
    create(ListIssueCommentsRequestSchema, {
      parent: issue.value.name,
      // Try to get all comments at once with max page size.
      pageSize: 1000,
    })
  );
};

watchEffect(prepareIssueComments);

const issueComments = computed(() => {
  return issueCommentStore.getIssueComments(issue.value.name);
});

const allowCreateComment = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.issueComments.create");
});

const cancelEditComment = () => {
  state.activeComment = undefined;
  state.editCommentMode = false;
  state.editComment = "";
};

const doCreateComment = async (comment: string) => {
  await issueCommentStore.createIssueComment({
    issueName: issue.value.name,
    comment,
  });
  state.newComment = "";
  await prepareIssueComments();
};

const allowEditIssueComment = (comment: IssueComment) => {
  const commentType = getIssueCommentType(comment);
  // Check if comment is user-editable
  const isEditable =
    commentType === IssueCommentType.USER_COMMENT ||
    (commentType === IssueCommentType.APPROVAL && comment.comment !== "");

  if (!isEditable) {
    return false;
  }
  if (currentUser.value.email === extractUserId(comment.creator)) {
    return true;
  }
  return hasProjectPermissionV2(project.value, "bb.issueComments.update");
};

const onUpdateComment = (issueComment: IssueComment) => {
  state.activeComment = issueComment;
  state.editCommentMode = true;
  state.editComment = issueComment.comment;
};

const doUpdateComment = async () => {
  if (!state.activeComment) {
    return;
  }
  if (!state.editComment) {
    return;
  }

  await issueCommentStore.updateIssueComment({
    issueCommentName: state.activeComment.name,
    comment: state.editComment,
  });
  cancelEditComment();
};

const allowUpdateComment = computed(() => {
  return state.editComment && state.editComment != state.activeComment!.comment;
});

const handleUpdateIssue = (updatedIssue: ComposedIssue) => {
  Object.assign(issue.value, updatedIssue);
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
