import { WORKSPACE_ID } from "./index";

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
    const attrs = this.normalizedRequestAttrs("bookmark-new");
    const newBookmark = {
      ...attrs,
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.creatorId,
      lastUpdatedTs: ts,
      workspaceId: WORKSPACE_ID,
    };
    return schema.bookmarks.create(newBookmark);
  });

  route.delete("/bookmark/:bookmarkId", function (schema, request) {
    return schema.bookmarks.find(request.params.bookmarkId).destroy();
  });
}
