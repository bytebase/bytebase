import { Factory } from "miragejs";
import faker from "faker";
import { UNKNOWN_ID } from "../../types";

export default {
  issue: Factory.extend({
    name(i) {
      return faker.fake("{{lorem.sentence}}");
    },
    creatorId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    updatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    status() {
      return "OPEN";
    },
    type() {
      return "bb.database.create";
    },
    description() {
      return faker.fake("{{lorem.paragraphs}}");
    },
    sql() {
      return "CREATE TABLE t1 (id INT NOT NULL);";
    },
    rollbackSql() {
      return "";
    },
    assigneeId() {
      return UNKNOWN_ID;
    },
    subscriberIdList() {
      return [];
    },
    payload() {
      return {
        // Requested Database name
        1: "Fake DB",
      };
    },
  }),
};
