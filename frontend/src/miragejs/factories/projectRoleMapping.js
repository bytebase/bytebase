import { Factory } from "miragejs";

export default {
  projectRoleMapping: Factory.extend({
    creatorId() {
      return "100";
    },
    updaterId() {
      return "200";
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    role() {
      let dice = Math.random();
      if (dice < 0.5) {
        return "OWNER";
      } else {
        return "DEVELOPER";
      }
    },
    principalId() {
      return "100";
    },
  }),
};
