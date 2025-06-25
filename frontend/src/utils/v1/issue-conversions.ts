import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Issue as OldIssue } from "@/types/proto/v1/issue_service";
import { Issue as OldIssueProto } from "@/types/proto/v1/issue_service";
import type { IssueComment as OldIssueComment } from "@/types/proto/v1/issue_service";
import { IssueComment as OldIssueCommentProto } from "@/types/proto/v1/issue_service";
import type { Issue as NewIssue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { IssueComment as NewIssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { IssueCommentSchema } from "@/types/proto-es/v1/issue_service_pb";

// Convert old proto Issue to proto-es Issue
export const convertOldIssueToNew = (oldIssue: OldIssue): NewIssue => {
  const json = OldIssueProto.toJSON(oldIssue) as any;
  return fromJson(IssueSchema, json);
};

// Convert proto-es Issue to old proto Issue
export const convertNewIssueToOld = (newIssue: NewIssue): OldIssue => {
  const json = toJson(IssueSchema, newIssue);
  return OldIssueProto.fromJSON(json);
};

// Convert old proto IssueComment to proto-es IssueComment
export const convertOldIssueCommentToNew = (oldComment: OldIssueComment): NewIssueComment => {
  const json = OldIssueCommentProto.toJSON(oldComment) as any;
  return fromJson(IssueCommentSchema, json);
};

// Convert proto-es IssueComment to old proto IssueComment
export const convertNewIssueCommentToOld = (newComment: NewIssueComment): OldIssueComment => {
  const json = toJson(IssueCommentSchema, newComment);
  return OldIssueCommentProto.fromJSON(json);
};