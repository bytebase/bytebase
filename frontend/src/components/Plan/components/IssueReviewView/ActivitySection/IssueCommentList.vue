<template>
  <div class="flex flex-col">
    <ul>
      <IssueDescriptionComment :issue-comments="issueComments" />
      <IssueCommentView
        v-for="(item, index) in issueComments"
        :key="item.name"
        class="group"
        :is-last="index === issueComments.length - 1"
        :issue-comment="item"
      >
        <template v-if="allowEditComment(item)" #subject-suffix>
          <NButton
            v-if="!commentEdit.state.editMode"
            quaternary
            size="tiny"
            class="text-gray-500 hover:text-gray-700"
            @click.prevent="commentEdit.startEditComment(item)"
          >
            <PencilIcon class="w-3.5 h-3.5" />
          </NButton>
        </template>

        <template v-if="item.comment" #comment>
          <EditableMarkdownContent
            :content="item.comment"
            :edit-content="commentEdit.state.editContent"
            :project="project"
            :is-editing="isEditingComment(item.name)"
            :allow-save="commentEdit.allowUpdateComment.value"
            @update:edit-content="commentEdit.state.editContent = $event"
            @save="commentEdit.saveEditComment"
            @cancel="commentEdit.cancelEditComment"
          />
        </template>
      </IssueCommentView>
    </ul>

    <div
      v-if="!commentEdit.state.editMode && commentEdit.allowCreateComment.value"
      class="mt-2"
    >
      <div class="flex gap-3">
        <div class="shrink-0 pt-1">
          <UserAvatar :user="commentEdit.currentUser.value" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="sr-only" id="issue-comment-editor"></h3>
          <MarkdownEditor
            mode="editor"
            :content="commentEdit.state.newComment"
            :project="project"
            :maxlength="65536"
            @change="(val: string) => (commentEdit.state.newComment = val)"
            @submit="handleCreateComment"
          />
          <div class="mt-3 flex items-center justify-end">
            <NButton
              type="primary"
              :disabled="commentEdit.state.newComment.length === 0"
              @click.prevent="handleCreateComment"
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
import { computed, onMounted, watch, watchEffect } from "vue";
import { useRoute } from "vue-router";
import MarkdownEditor from "@/components/MarkdownEditor";
import { usePlanContext } from "@/components/Plan/logic";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { useCurrentProjectV1, useIssueCommentStore } from "@/store";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { IssueCommentView, useCommentEdit } from "./IssueCommentView";
import EditableMarkdownContent from "./IssueCommentView/EditableMarkdownContent.vue";
import IssueDescriptionComment from "./IssueCommentView/IssueDescriptionComment.vue";

const route = useRoute();
const { project } = useCurrentProjectV1();
const { events } = usePlanContext();
const issueCommentStore = useIssueCommentStore();

const commentEdit = useCommentEdit(project);
const { issueName, allowEditComment } = commentEdit;

const isEditingComment = (commentName: string): boolean => {
  return (
    commentEdit.state.editMode &&
    commentEdit.state.activeComment?.name === commentName
  );
};

const fetchIssueComments = async () => {
  if (!issueName.value) return;
  await issueCommentStore.listIssueComments(
    create(ListIssueCommentsRequestSchema, {
      parent: issueName.value,
      pageSize: 1000,
    })
  );
};

watchEffect(fetchIssueComments);

// Refresh comments when a review or status action is performed
events.on("perform-issue-review-action", fetchIssueComments);
events.on("perform-issue-status-action", fetchIssueComments);

const issueComments = computed(() => {
  if (!issueName.value) return [];
  return issueCommentStore.getIssueComments(issueName.value);
});

const handleCreateComment = async () => {
  await commentEdit.createComment(commentEdit.state.newComment);
  await fetchIssueComments();
};

onMounted(() => {
  watch(
    () => route.hash,
    (hash) => {
      if (hash.match(/^#activity(\d+)/)) {
        const elem =
          document.querySelector(hash) || document.querySelector("#activity");
        setTimeout(() => elem?.scrollIntoView());
      }
    },
    { immediate: true }
  );
});
</script>
