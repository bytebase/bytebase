import { Factory } from "miragejs";
import faker from "faker";
import { UNKNOWN_ID } from "../../types";

export default {
  bookmark: Factory.extend({
    creatorId() {
      return UNKNOWN_ID;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    updatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
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
