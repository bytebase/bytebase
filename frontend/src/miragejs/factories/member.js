import { Factory } from "miragejs";

export default {
  member: Factory.extend({
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    role() {
      let dice = Math.random();
      if (dice < 0.33) {
        return "OWNER";
      } else if (dice < 0.66) {
        return "DBA";
      } else {
        return "DEVELOPER";
      }
    },
    principalId() {
      return "100";
    },
    updaterId() {
      return "200";
    },
  }),
};
