import { FAKE_API_CALLER_ID, WORKSPACE_ID } from "./index";

export default function configureUser(route) {
  route.get("/user/:id");

  route.get("/user");

  route.post("/user", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user-new");
    const user = schema.users.findBy({ email: attrs.email });
    if (user) {
      return user;
    }

    const ts = Date.now();
    return schema.users.create({
      ...attrs,
      creatorId: FAKE_API_CALLER_ID,
      updaterId: FAKE_API_CALLER_ID,
      createdTs: ts,
      lastUpdatedTs: ts,
    });
  });

  route.patch("/user/:userId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user-patch");
    const user = schema.users.find(request.params.userId);

    if (!user) {
      return new Response(
        404,
        {},
        {
          errors: "User id " + request.params.userId + " not found",
        }
      );
    }

    const ts = Date.now();
    return user.update({
      ...attrs,
      updaterId: FAKE_API_CALLER_ID,
      lastUpdatedTs: ts,
    });
  });
}
