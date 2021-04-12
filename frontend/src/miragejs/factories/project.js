import { Factory } from "miragejs";
import { UNKNOWN_ID } from "../../types";

export default {
  project: Factory.extend({
    rowStatus(i) {
      return i > 3 ? "ARCHIVED" : "NORMAL";
    },
    creatorId() {
      return UNKNOWN_ID;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    name(i) {
      return "project " + i;
    },
  }),
};
