import { Factory } from "miragejs";
import faker from "faker";

export default {
  task: Factory.extend({
    name(i) {
      return faker.fake("{{lorem.sentence}}");
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    status() {
      return "OPEN";
    },
    category() {
      let dice = Math.random();
      if (dice < 0.33) {
        return "DDL";
      } else if (dice < 0.66) {
        return "DML";
      } else {
        return "OPS";
      }
    },
    type() {
      return "bytebase.datasource.create";
    },
    content() {
      return faker.fake("{{lorem.paragraphs}}");
    },
    stageProgressList() {
      return [
        {
          stageId: "1",
          stageName: "Stage Foo",
          status: "DONE",
        },
      ];
    },
    creator() {
      return {
        id: "100",
        name: "John Appleseed",
      };
    },
    assignee() {
      return {
        id: "200",
        name: "Jim Gray",
      };
    },
    subscriberIdList() {
      return new Array();
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
