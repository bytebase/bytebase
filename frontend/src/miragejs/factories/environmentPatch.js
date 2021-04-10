import { Factory } from "miragejs";

export default {
  environmentPatch: Factory.extend({
    rowStatus() {
      return "ARCHIVED";
    },
    name() {
      return "Updated environment";
    },
  }),
};
