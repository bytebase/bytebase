import { Factory } from "miragejs";
import faker from "faker";

export default {
  user: Factory.extend({
    name() {
      return faker.fake("{{name.findName}}");
    },
    email() {
      return faker.fake("{{internet.email}}");
    },
    passwordHash() {
      return faker.fake("{{internet.password}}");
    },
    mobile() {
      return faker.fake("{{phone.phoneNumber}}");
    },
  }),
};
