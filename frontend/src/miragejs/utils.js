import { WORKSPACE_ID } from "./routes/index";

export const randomNumber = (quantity) =>
  Math.floor(Math.random() * quantity) + 1;

export const postMessageToOwnerAndDBA = function (
  schema,
  creatorId,
  messageTemplate
) {
  const messageList = [];
  const allOwnerAndDBAs = schema.roleMappings.where((roleMapping) => {
    return (
      roleMapping.workspaceId == WORKSPACE_ID &&
      (roleMapping.role == "OWNER" || roleMapping.role == "DBA")
    );
  }).models;

  allOwnerAndDBAs.forEach((roleMapping) => {
    messageList.push({
      ...messageTemplate,
      receiverId: roleMapping.principalId,
    });
  });

  for (const message of messageList) {
    // We only send out message if it's NOT destined to self.
    if (creatorId != message.receiverId) {
      schema.messages.create(message);
    }
  }
};
