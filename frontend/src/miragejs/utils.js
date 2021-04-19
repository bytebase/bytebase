import { UNKNOWN_ID } from "../types";
import { WORKSPACE_ID } from "./routes/index";

export const randomNumber = (quantity) =>
  Math.floor(Math.random() * quantity) + 1;

export const postMessageToOwnerAndDBA = function (
  schema,
  creatorId,
  messageTemplate
) {
  const messageList = [];
  const allOwnerAndDBAs = schema.members.where((member) => {
    return (
      member.workspaceId == WORKSPACE_ID &&
      (member.role == "OWNER" || member.role == "DBA")
    );
  }).models;

  allOwnerAndDBAs.forEach((member) => {
    messageList.push({
      ...messageTemplate,
      receiverId: member.principalId,
    });
  });

  for (const message of messageList) {
    // We only send out message if it's NOT destined to self.
    if (creatorId != message.receiverId) {
      schema.messages.create(message);
    }
  }
};

export const postIssueMessageToReceiver = function (
  schema,
  issue,
  creatorId,
  messageTemplate
) {
  const messageList = [];

  messageList.push({
    ...messageTemplate,
    receiverId: issue.creatorId,
  });

  if (issue.assigneeId != UNKNOWN_ID && issue.assigneeId != issue.creatorId) {
    messageList.push({
      ...messageTemplate,
      receiverId: issue.assigneeId,
    });
  }

  for (let subscriberId of issue.subscriberIdList) {
    if (subscriberId != issue.creatorId && subscriberId != issue.assigneeId) {
      messageList.push({
        ...messageTemplate,
        receiverId: subscriberId,
      });
    }
  }

  for (const message of messageList) {
    // We only send out message if it's NOT destined to self.
    if (creatorId != message.receiverId) {
      schema.messages.create(message);
    }
  }
};
