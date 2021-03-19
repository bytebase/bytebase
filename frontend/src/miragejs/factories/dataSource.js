import { Factory } from "miragejs";

export default {
  dataSource: Factory.extend({
    name(i) {
      return "ds" + i;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    type(i) {
      if (i % 2 == 0) {
        return "RW";
      }
      if (i % 2 == 1) {
        return "RO";
      }
    },
    username() {
      return "root";
    },
    password(i) {
      return "pwd" + i;
    },
    afterCreate(dataSource, server) {
      server.create("dataSourceMember", {
        workspaceId: dataSource.workspaceId,
        dataSource,
        principalId: "1",
        taskId: "2",
      });

      if (dataSource.id % 2 == 0) {
        server.create("dataSourceMember", {
          workspaceId: dataSource.workspaceId,
          dataSource,
          principalId: "2",
          taskId: "3",
        });
      }

      if (dataSource.id % 4 == 0) {
        server.create("dataSourceMember", {
          workspaceId: dataSource.workspaceId,
          dataSource,
          principalId: "3",
        });
      }
    },
  }),
};
