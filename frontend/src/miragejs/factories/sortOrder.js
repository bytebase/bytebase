import { Factory } from "miragejs";
import faker from "faker";

export default {
  sortOrder: Factory.extend({
    from() {
      return 0;
    },
    to() {
      return 1;
    },
  }),
};
