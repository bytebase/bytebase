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
      if (i % 3 == 0) {
        return "ADMIN";
      }
      if (i % 3 == 1) {
        return "READWRITE";
      }
      if (i % 3 == 2) {
        return "READONLY";
      }
    },
    username() {
      return "root";
    },
    password(i) {
      return "pwd" + i;
    },
  }),
};
