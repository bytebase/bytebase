import { Factory } from "miragejs";

export default {
  messagePatch: Factory.extend({
    status() {
      return "CONSUMED";
    },
  }),
};
