import { Factory } from "miragejs";
import faker from "faker";
import { UNKNOWN_ID } from "../../types";

export default {
  user: Factory.extend({
    creatorId() {
      return UNKNOWN_ID;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    name() {
      return faker.fake("{{name.findName}}");
    },
    status() {
      return "INVITED";
    },
    email() {
      return faker.fake("{{internet.email}}");
    },
    passwordHash() {
      return faker.fake("{{internet.password}}");
    },
  }),
};
