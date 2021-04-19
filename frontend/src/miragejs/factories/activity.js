import { Factory } from "miragejs";
import { UNKNOWN_ID } from "../../types";

export default {
  activity: Factory.extend({
    creatorId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    updatedTs(i) {
      return Date.now() - (10 - i) * 1800 * 1000;
    },
    actionType() {
      return "bytebase.task.comment.create";
    },
    containerId() {
      return "0";
    },
    comment() {
      return "";
    },
  }),
};
