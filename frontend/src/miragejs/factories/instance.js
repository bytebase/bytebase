import { Factory } from "miragejs";

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
      server.create("dataSource", {
        instance,
        name: instance.name + " admin ds1",
        type: "RW",
      });

      for (let i = 0; i < 3; i++) {
        const dbName = "db" + (i + 1);
        const database = server.create("database", {
          instance,
          name: dbName,
        });

        server.create("dataSource", {
          instance,
          database,
          name: dbName + " rw ds2",
          type: "RW",
          username: "rootRW",
          password: "pwdRW",
        });

        server.create("dataSource", {
          instance,
          database,
          name: dbName + " ro ds3",
          type: "RO",
          username: "rootRO",
          password: "pwdRO",
        });

        server.create("dataSource", {
          instance,
          database,
          name: dbName + " ro ds4",
          type: "RO",
          username: "rootRO",
          password: "pwdRO",
        });
      }
    },
  }),
};
