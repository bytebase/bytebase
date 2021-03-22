import { Factory } from "miragejs";
import faker from "faker";

export default {
  taskPatch: Factory.extend({
    updaterId() {
      return "100";
    },
    name(i) {
      return faker.fake("{{lorem.sentence}}");
    },
    status() {
      return "RUNNING";
    },
    description() {
      return faker.fake("{{lorem.paragraphs}}");
    },
    assigneeId() {
      return "200";
    },
    stageProgress() {
      return [
        {
          id: "1",
          status: "RUNNING",
        },
      ];
    },
    comment() {
      return "my update comment";
    },
    payload() {
      return {
        // Requested Database name
        1: "Mydb",
      };
    },
  }),
};
