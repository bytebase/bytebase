import { Factory } from "miragejs";

export default {
  activity: Factory.extend({
    createdTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    actionType() {
      return "bytebase.task.comment.create";
    },
    containerId() {
      return "0";
    },
    creatorId() {
      return "100";
    },
    payload() {
      return {};
    },
  }),
};
