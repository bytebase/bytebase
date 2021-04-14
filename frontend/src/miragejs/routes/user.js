import { WORKSPACE_ID } from "./index";

export default function configureUser(route) {
  route.get("/user/:id");

  route.get("/user");

  route.post("/user", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user-new");
    const user = schema.users.findBy({ email: attrs.email });
    if (user) {
      return user;
    }
    return schema.users.create(attrs);
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

    return user.update(attrs);
  });
}
