import { Response } from "miragejs";

export default function configureDataSource(route) {
  route.get("/database/:databaseId/datasource", function (schema, request) {
    const database = schema.databases.find(request.params.databaseId);
    if (database) {
      return schema.dataSources.where((dataSource) => {
        return (
          dataSource.databaseId == database.id && dataSource.type != "ADMIN"
        );
      });
    }
    return new Response(
      404,
      {},
      { errors: "Database " + request.params.databaseId + " not found" }
    );
  });

  route.get("/database/:databaseId/datasource/:id", function (schema, request) {
    const database = schema.databases.find(request.params.databaseId);
    if (database) {
      const dataSource = schema.dataSources.find(request.params.id);
      if (dataSource) {
        return dataSource;
      }
      return new Response(
        404,
        {},
        { errors: "Data Source " + request.params.id + " not found" }
      );
    }
    return new Response(
      404,
      {},
      { errors: "Database " + request.params.databaseId + " not found" }
    );
  });

  route.post("/database/:databaseId/datasource", function (schema, request) {
    const database = schema.databases.find(request.params.databaseId);
    if (database) {
      const ts = Date.now();
      const attrs = this.normalizedRequestAttrs("data-source-new");
      const newDataSource = {
        ...attrs,
        creatorId: attrs.creatorId,
        createdTs: ts,
        updaterId: attrs.creatorId,
        updatedTs: ts,
      };
      return schema.dataSources.create(newDataSource);
    }
    return new Response(
      404,
      {},
      { errors: "Dnstance " + request.params.databaseId + " not found" }
    );
  });

  route.patch(
    "/database/:databaseId/datasource/:id",
    function (schema, request) {
      const database = schema.databases.find(request.params.databaseId);
      if (database) {
        const dataSource = schema.dataSources.find(request.params.id);
        if (!dataSource) {
          return new Response(
            404,
            {},
            {
              errors: "Data source " + request.params.id + " not found",
            }
          );
        }
        const attrs = this.normalizedRequestAttrs("data-source");

        let hasChange = false;

        if (attrs.name) {
          if (attrs.name != dataSource.name) {
            hasChange = true;
          }
        }

        if (attrs.username) {
          if (attrs.username != dataSource.username) {
            hasChange = true;
          }
        }

        if (attrs.password) {
          if (attrs.password != dataSource.password) {
            hasChange = true;
          }
        }

        if (hasChange) {
          return dataSource.update({ ...attrs, updatedTs: Date.now() });
        }
        return dataSource;
      }
      return new Response(
        404,
        {},
        { errors: "Database " + request.params.databaseId + " not found" }
      );
    }
  );

  route.delete(
    "/database/:databaseId/datasource/:id",
    function (schema, request) {
      const database = schema.instances.find(request.params.databaseId);
      if (database) {
        const dataSource = schema.dataSources.find(request.params.id);
        if (!dataSource) {
          return new Response(
            404,
            {},
            {
              errors: "Data source " + request.params.id + " not found",
            }
          );
        }
        const dataSourceMemberList = schema.dataSourceMembers.where(
          (dataSourceMember) => {
            return dataSourceMember.dataSourceId == dataSource.id;
          }
        );
        dataSourceMemberList.models.forEach((member) => member.destroy());
        return dataSource.destroy();
      }
      return new Response(
        404,
        {},
        { errors: "Database " + request.params.databaseId + " not found" }
      );
    }
  );

  // Data Source Member
  // Be careful to use :dataSourceId instead of :id otherwise this.normalizedRequestAttrs
  // would de-serialize to id, which would prevent auto increment id logic.
  route.post(
    "/database/:databaseId/datasource/:dataSourceId/member",
    function (schema, request) {
      const database = schema.databases.find(request.params.databaseId);
      if (database) {
        const dataSource = schema.dataSources.find(request.params.dataSourceId);
        if (dataSource) {
          const attrs = this.normalizedRequestAttrs(
            "data-source-member-create"
          );
          const newList = dataSource.memberList;
          const member = newList.find(
            (item) => item.principalId == attrs.principalId
          );
          if (!member) {
            newList.push({
              principalId: attrs.principalId,
              issueId: attrs.issueId,
              createdTs: Date.now(),
            });
            return dataSource.update({
              memberList: newList,
              updaterId: attrs.creatorId,
            });
          }
          return dataSource;
        }
        return new Response(
          404,
          {},
          {
            errors: "Data source " + request.params.dataSourceId + " not found",
          }
        );
      }
      return new Response(
        404,
        {},
        { errors: "Database " + request.params.databaseId + " not found" }
      );
    }
  );

  route.delete(
    "/database/:databaseId/datasource/:dataSourceId/member/:memberId",
    function (schema, request) {
      const database = schema.databases.find(request.params.databaseId);
      if (database) {
        const dataSource = schema.dataSources.find(request.params.dataSourceId);
        if (!dataSource) {
          return new Response(
            404,
            {},
            {
              errors:
                "Data source " + request.params.dataSourceId + " not found",
            }
          );
        }
        const newList = dataSource.memberList;
        const index = newList.findIndex(
          (item) => item.principalId == request.params.memberId
        );
        if (index >= 0) {
          newList.splice(index, 1);
          return dataSource.update({
            memberList: newList,
          });
        }
        return dataSource;
      }
      return new Response(
        404,
        {},
        { errors: "Database " + request.params.databaseId + " not found" }
      );
    }
  );
}
