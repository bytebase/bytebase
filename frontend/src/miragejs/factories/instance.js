import { Factory } from "miragejs";
import faker from "faker";

export default {
  instance: Factory.extend({
    name(i) {
      if (i == 0) {
        return "dev env";
      } else if (i == 1) {
        return "test env";
      } else if (i == 2) {
        return "staging env";
      } else {
        return "prod env " + i;
      }
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    externalLink() {
      return "google.com";
    },
    host(i) {
      if (i == 0) {
        return "localhost";
      } else if (i == 1) {
        return "127.0.0.1";
      } else if (i == 2) {
        return "13.24.32.122";
      } else {
        return "mydb.com";
      }
    },
    port(i) {
      if (i == 0) {
        return "3306";
      } else if (i == 1) {
        return "";
      } else if (i == 2) {
        return "15202";
      } else {
        return "5432";
      }
    },
    afterCreate(instance, server) {
      const allDatabase = server.create("database", {
        workspaceId: instance.workspaceId,
        instance,
        name: "*",
        // Demo Owner
        ownerId: "1",
      });

      server.create("dataSource", {
        workspaceId: instance.workspaceId,
        instance,
        database: allDatabase,
        name: instance.name + " admin ds1",
        type: "RW",
        username: "adminRW",
        password: "pwdadminRW",
      });

      server.create("dataSource", {
        workspaceId: instance.workspaceId,
        instance,
        database: allDatabase,
        name: instance.name + " admin ds2",
        type: "RO",
        username: "adminRO",
        password: "pwdadminRO",
      });

      for (let i = 0; i < 3; i++) {
        const database = server.create("database", {
          workspaceId: instance.workspaceId,
          instance,
          ownerId: ((i % 3) + 1).toString(),
        });

        server.create("dataSource", {
          workspaceId: instance.workspaceId,
          instance,
          database,
          name: database.name + " rw ds3",
          type: "RW",
          username: "rootRW",
          password: "pwdRW",
        });

        server.create("dataSource", {
          workspaceId: instance.workspaceId,
          instance,
          database,
          name: database.name + " ro ds4",
          type: "RO",
          username: "rootRO",
          password: "pwdRO",
        });
      }
    },
  }),
};
