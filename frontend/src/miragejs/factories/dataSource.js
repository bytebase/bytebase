import { Factory } from "miragejs";
import { UNKNOWN_ID } from "../../types";

export default {
  dataSource: Factory.extend({
    name(i) {
      return "ds" + i;
    },
    creatorId(i) {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    updaterId(i) {
      return UNKNOWN_ID;
    },
    updatedTs(i) {
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
    memberList() {
      return [];
    },
  }),
};
