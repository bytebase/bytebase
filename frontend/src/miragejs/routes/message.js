import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";

export default function configureMessage(route) {
  route.get("/message", function (schema, request) {
    const {
      queryParams: { user: userId },
    } = request;
    return schema.messages
      .where((message) => {
        return (
          message.workspaceId == WORKSPACE_ID && message.receiverId == userId
        );
      })
      .sort((a, b) => b.createdTs - a.createdTs);
  });

  route.patch("/message/:messageId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("message-patch");
    const message = schema.messages.find(request.params.messageId);
    if (message) {
      return message.update({ ...attrs, updatedTs: Date.now() });
    }
    return new Response(
      404,
      {},
      { errors: "Message " + request.params.messageId + " not found" }
    );
  });
}
