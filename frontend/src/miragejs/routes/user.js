import { WORKSPACE_ID } from "./index";

export default function configureUser(route) {
  route.get("/user/:id");

  route.get("/user");

  route.post("/user", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user");
    const user = schema.users.findBy({ email: attrs.email });
    if (user) {
      return user;
    }
    return schema.users.create(attrs);
  });

  route.patch("/user/:userId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user");
    return schema.users.find(request.params.userId).update(attrs);
  });

  route.get("/user/:userId/database", function (schema, request) {
    return schema.databases.where((database) => {
      const dataSourceList = schema.dataSources.where({
        workspaceId: WORKSPACE_ID,
        databaseId: database.id,
      });

      for (const dataSource of dataSourceList.models) {
        if (
          dataSource.memberList.find((item) => {
            return item.principalId == request.params.userId;
          })
        ) {
          return true;
        }
      }
      return false;
    });
  });
}
