<template>
  <div class="flex flex-col gap-y-4">
    <ul>
      <IssueCreatedComment :issue-comments="issueComments" :issue="issue" />
      <IssueCommentView
        class="group"
        v-for="(item, index) in issueComments"
        :key="item.comment.name"
        :issue="issue"
        :index="index"
        :is-last="index === issueComments.length - 1"
        :issue-comment="item.comment"
        :similar="item.similar"
      >
        <template v-if="allowEditIssueComment(item.comment)" #subject-suffix>
          <div
            class="invisible group-hover:visible space-x-2 flex items-center text-control-light"
          >
            <div
              v-if="!state.editCommentMode"
              class="mr-2 flex items-center space-x-2"
            >
              <!-- Edit Comment Button-->
              <NButton
                quaternary
                size="tiny"
                @click.prevent="onUpdateComment(item.comment)"
              >
                <PencilIcon class="w-4 h-4 text-control-light" />
              </NButton>
            </div>
          </div>
        </template>

        <template #comment>
          <MarkdownEditor
            v-if="item.comment.comment"
            :mode="
              state.editCommentMode &&
              state.activeComment?.name === item.comment.name
                ? 'editor'
                : 'preview'
            "
            :content="item.comment.comment"
            :issue-list="issueList"
            :project="project"
            @change="(val: string) => (state.editComment = val)"
            @submit="doUpdateComment"
            @cancel="cancelEditComment"
          />
          <div
            v-if="
              state.editCommentMode &&
              state.activeComment?.name === item.comment.name
            "
            class="flex space-x-2 mt-4 items-center justify-end"
          >
            <NButton quaternary @click.prevent="cancelEditComment">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              :disabled="!allowUpdateComment"
              @click.prevent="doUpdateComment"
            >
              {{ $t("common.save") }}
            </NButton>
          </div>
        </template>
      </IssueCommentView>
    </ul>

    <div v-if="!state.editCommentMode && allowCreateComment">
      <div class="flex">
        <div class="flex-shrink-0">
          <div class="relative">
            <UserAvatar :user="currentUser" />
            <span
              class="absolute -bottom-0.5 -right-1 bg-white rounded-tl px-0.5 py-px"
            >
              <heroicons-solid:chat-alt
                class="h-3.5 w-3.5 text-control-light"
              />
            </span>
          </div>
        </div>
        <div class="ml-3 min-w-0 flex-1">
          <h3 class="sr-only" id="issue-comment-editor"></h3>
          <label for="comment" class="sr-only">
            {{ $t("issue.add-a-comment") }}
          </label>
          <MarkdownEditor
            mode="editor"
            :content="state.newComment"
            :issue-list="issueList"
            :project="project"
            @change="(val: string) => (state.newComment = val)"
            @submit="doCreateComment(state.newComment)"
          />
          <div class="my-3 flex items-center justify-between">
            <NButton
              size="small"
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
import { computed, onMounted, reactive, ref, watch, watchEffect } from "vue";
import { useRoute } from "vue-router";
import MarkdownEditor from "@/components/MarkdownEditor";
import UserAvatar from "@/components/User/UserAvatar.vue";
import {
  useCurrentUserV1,
  useIssueCommentStore,
  useIssueV1Store,
  useCurrentProjectV1,
  extractUserId,
} from "@/store";
import { isValidProjectName } from "@/types";
import type { ComposedIssue } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useIssueContext } from "../../logic";
import {
  IssueCommentView,
  isSimilarIssueComment,
  isUserEditableComment,
  type DistinctIssueComment,
} from "./IssueCommentView";
import IssueCreatedComment from "./IssueCommentView/IssueCreatedComment.vue";

interface LocalState {
  editCommentMode: boolean;
  activeComment?: IssueComment;
  editComment: string;
  newComment: string;
}

const route = useRoute();

const { project } = useCurrentProjectV1();
const { issue } = useIssueContext();
const issueList = ref<ComposedIssue[]>([]);

const state = reactive<LocalState>({
  editCommentMode: false,
  editComment: "",
  newComment: "",
});

const currentUser = useCurrentUserV1();
const issueV1Store = useIssueV1Store();
const issueCommentStore = useIssueCommentStore();

const prepareIssueListForMarkdownEditor = async () => {
  const project = issue.value.project;
  issueList.value = [];
  if (!isValidProjectName(project)) return;

  const list = await issueV1Store.listIssues({
    find: {
      project,
      query: "",
    },
  });
  issueList.value = list.issues;
};

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

const issueComments = computed((): DistinctIssueComment[] => {
  const list = issueCommentStore.getIssueComments(issue.value.name);
  const distinctIssueComments: DistinctIssueComment[] = [];
  for (let i = 0; i < list.length; i++) {
    const comment = list[i];
    if (distinctIssueComments.length === 0) {
      distinctIssueComments.push({ comment, similar: [] });
      continue;
    }

    const prev = distinctIssueComments[distinctIssueComments.length - 1];
    if (isSimilarIssueComment(prev.comment, comment)) {
      prev.similar.push(comment);
    } else {
      distinctIssueComments.push({ comment, similar: [] });
    }
  }
  return distinctIssueComments;
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
  if (!isUserEditableComment(comment)) {
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

watch(
  () => issue.value.project,
  () => {
    prepareIssueListForMarkdownEditor();
  },
  { immediate: true }
);
</script>
