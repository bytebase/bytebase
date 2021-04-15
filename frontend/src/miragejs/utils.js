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
