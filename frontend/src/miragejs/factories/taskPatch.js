import { Factory } from "miragejs";
import faker from "faker";

export default {
  taskPatch: Factory.extend({
    producer() {
      return {
        id: "100",
        name: "Ed Codd",
      };
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
    stageProgressList() {
      return [
        {
          id: "1",
          status: "RUNNING",
        },
      ];
    },
    payload() {
      return {
        // Requested Database name
        1: "Mydb",
      };
    },
  }),
};
