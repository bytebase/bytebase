import { Factory } from "miragejs";
import faker from "faker";

export default {
  message: Factory.extend({
    createdTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    type() {
      return "bb.msg.task.assign";
    },
    status(i) {
      return Math.floor(Math.random() * 2) % 2 == 0 ? "DELIVERED" : "CONSUMED";
    },
    containerId() {
      return "0";
    },
    creatorId() {
      return "100";
    },
    receiverId() {
      return "200";
    },
    description(i) {
      return Math.floor(Math.random() * 3) % 3 == 0
        ? ""
        : faker.fake("{{lorem.sentences}}");
    },
    payload() {
      return {
        taskName: faker.fake("{{lorem.sentence}}"),
      };
    },
  }),
};
