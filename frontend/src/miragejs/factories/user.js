/*
 * Mirage JS guide on Factories: https://miragejs.com/docs/data-layer/factories
 */
import { Factory } from "miragejs";

/*
 * Faker Github repository: https://github.com/Marak/Faker.js#readme
 */
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
