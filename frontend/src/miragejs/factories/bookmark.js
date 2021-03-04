import { Factory } from "miragejs";
import faker from "faker";

export default {
  bookmark: Factory.extend({
    name() {
      return "bookmark " + faker.fake("{{lorem.word}}");
    },
    link(i) {
      return "favlink" + (i + 1);
    },
    creatorId() {
      return "100";
    },
  }),
};
