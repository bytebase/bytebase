import { Factory } from "miragejs";

export default {
  environment: Factory.extend({
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
