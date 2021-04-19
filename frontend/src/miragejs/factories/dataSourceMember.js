import { Factory } from "miragejs";

export default {
  dataSourceMember: Factory.extend({
    principalId(i) {
      return "100";
    },
    issueId(i) {
      return (i + 1).toString();
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
  }),
};
