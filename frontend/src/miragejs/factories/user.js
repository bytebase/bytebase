import { Factory } from "miragejs";
import faker from "faker";

export default {
  user: Factory.extend({
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
