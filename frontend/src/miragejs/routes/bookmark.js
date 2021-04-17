import { FAKE_API_CALLER_ID, WORKSPACE_ID } from "./index";

export default function configureBookmark(route) {
  route.get("/bookmark", function (schema, request) {
    const {
      queryParams: { user: userId },
    } = request;

    return schema.bookmarks.where({
      workspaceId: WORKSPACE_ID,
      creatorId: userId,
    });
  });

  route.post("/bookmark", function (schema, request) {
    const ts = Date.now();
    const newBookmark = {
      ...this.normalizedRequestAttrs("bookmark"),
      creatorId: FAKE_API_CALLER_ID,
      updaterId: FAKE_API_CALLER_ID,
      createdTs: ts,
      updatedTs: ts,
      workspaceId: WORKSPACE_ID,
    };
    return schema.bookmarks.create(newBookmark);
  });

  route.patch("/bookmark/:bookmarkId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("bookmark");
    return schema.bookmarks.find(request.params.bookmarkId).update(attrs);
  });

  route.delete("/bookmark/:bookmarkId", function (schema, request) {
    return schema.bookmarks.find(request.params.bookmarkId).destroy();
  });
}
