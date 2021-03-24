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
    afterCreate(dataSource, server) {},
    memberList(i) {
      const list = [
        {
          principalId: "1",
          taskId: (i + 1).toString(),
          createdTs: Date.now() - (i + 1) * 1800 * 1000,
        },
      ];
      if (i % 2 == 0) {
        list.push({
          principalId: "2",
          taskId: (i + 1).toString(),
          createdTs: Date.now() - (i + 1) * 1800 * 1000,
        });
      }
      if (i % 4 == 0) {
        list.push({
          principalId: "3",
          taskId: (i + 1).toString(),
          createdTs: Date.now() - (i + 1) * 1800 * 1000,
        });
      }
      return list;
    },
  }),
};
