import { Factory } from "miragejs";

export default {
  environment: Factory.extend({
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
    rowStatus(i) {
      return i > 3 ? "ARCHIVED" : "NORMAL";
    },
    name(i) {
      if (i == 0) {
        return "Sandbox A";
      } else if (i == 1) {
        return "Integration";
      } else if (i == 2) {
        return "Staging";
      } else if (i == 3) {
        return "Prod";
      } else {
        return "Archived Env " + (i - 3);
      }
    },
    order(i) {
      return i;
    },
  }),
};
