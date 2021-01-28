import { Factory } from "miragejs";
import faker from "faker";

export default {
  pipeline: Factory.extend({
    name(i) {
      return faker.fake("{{lorem.sentence}}");
    },
    createdTs(i) {
      const scaleFactor = Math.random() * i;
      return Date.now() - scaleFactor * 3600 * 6 * 1000;
    },
    lastUpdatedTs(i) {
      const scaleFactor = Math.random() * i;
      return Date.now() - scaleFactor * 3600 * 20 * 1000;
    },
    status(i) {
      if (i % 5 == 0) {
        return "CREATED";
      } else if (i % 5 == 1) {
        return "RUNNING";
      } else if (i % 5 == 2) {
        return "DONE";
      } else if (i % 5 == 3) {
        return "FAILED";
      } else {
        return "CANCELED";
      }
    },
    type() {
      let dice = Math.random();
      if (dice < 0.33) {
        return "DDL";
      } else if (dice < 0.66) {
        return "DML";
      } else {
        return "OTHER";
      }
    },
    currentStageId() {
      return "1";
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
  }),
};
