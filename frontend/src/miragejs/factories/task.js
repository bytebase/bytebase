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
    type() {
      return "bytebase.database.request";
    },
    description() {
      return faker.fake("{{lorem.paragraphs}}");
    },
    stageProgressList() {
      return [
        {
          id: "1",
          name: "Stage Foo",
          status: "DONE",
        },
      ];
    },
    creatorId() {
      return "100";
    },
    assigneeId() {
      return "200";
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
