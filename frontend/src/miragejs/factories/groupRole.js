import { randomNumber } from "./utils";

/*
 * Mirage JS guide on Factories: https://miragejs.com/docs/data-layer/factories
 */
import { Factory, trait, association } from "miragejs";

/*
 * Faker Github repository: https://github.com/Marak/Faker.js#readme
 */
import faker from "faker";

export default {
  groupRole: Factory.extend({
    role() {
      let dice = Math.random();
      if (dice < 0.5) {
        return "OWNER";
      } else {
        return "DEVELOPER";
      }
    },

    group: association(),

    user: association(),

    afterCreate(groupRole, server) {},
  }),
};
