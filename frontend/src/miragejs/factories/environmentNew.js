import { Factory } from "miragejs";

export default {
  environmentNew: Factory.extend({
    name() {
      return "New environment";
    },
  }),
};
