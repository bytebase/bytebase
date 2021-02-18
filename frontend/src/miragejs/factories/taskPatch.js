import { Factory } from "miragejs";
import faker from "faker";

export default {
  taskPatch: Factory.extend({
    name(i) {
      return faker.fake("{{lorem.sentence}}");
    },
    status() {
      return "RUNNING";
    },
    content() {
      return faker.fake("{{lorem.paragraphs}}");
    },
    assignee() {
      return {
        id: "200",
        name: "Jim Gray",
      };
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
        type: "bytebase.datasource.create",
        fieldList: {
          // Requested Database name
          1: "Mydb",
        },
      };
    },
  }),
};
