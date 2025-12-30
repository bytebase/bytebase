import type { ComputedRef, Ref } from "vue";
import { computed, reactive } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import {
  extractUserId,
  getIssueCommentType,
  IssueCommentType,
  useCurrentUserV1,
  useIssueCommentStore,
} from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

interface CommentEditState {
  editMode: boolean;
  activeComment?: IssueComment;
  editContent: string;
  newComment: string;
}

export function useCommentEdit(project: Ref<Project> | ComputedRef<Project>) {
  const { plan, issue } = usePlanContext();
  const currentUser = useCurrentUserV1();
  const issueCommentStore = useIssueCommentStore();

  const issueName = computed(() => issue.value?.name || plan.value.issue);

  const state = reactive<CommentEditState>({
    editMode: false,
    editContent: "",
    newComment: "",
  });

  const allowCreateComment = computed(() => {
    return hasProjectPermissionV2(project.value, "bb.issueComments.create");
  });

  const allowEditComment = (comment: IssueComment): boolean => {
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

  const allowUpdateComment = computed(() => {
    return (
      !!state.editContent && state.editContent !== state.activeComment?.comment
    );
  });

  const startEditComment = (comment: IssueComment) => {
    state.activeComment = comment;
    state.editMode = true;
    state.editContent = comment.comment;
  };

  const cancelEditComment = () => {
    state.activeComment = undefined;
    state.editMode = false;
    state.editContent = "";
  };

  const saveEditComment = async () => {
    if (!state.activeComment || !state.editContent) {
      return;
    }

    await issueCommentStore.updateIssueComment({
      issueCommentName: state.activeComment.name,
      comment: state.editContent,
    });
    cancelEditComment();
  };

  const createComment = async (comment: string) => {
    if (!issueName.value) return;
    await issueCommentStore.createIssueComment({
      issueName: issueName.value,
      comment,
    });
    state.newComment = "";
  };

  return {
    state,
    issueName,
    currentUser,
    allowCreateComment,
    allowEditComment,
    allowUpdateComment,
    startEditComment,
    cancelEditComment,
    saveEditComment,
    createComment,
  };
}
