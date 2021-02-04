/*
 * Mirage JS guide on Factories: https://miragejs.com/docs/data-layer/factories
 */
import { Factory, association } from "miragejs";

/*
 * Faker Github repository: https://github.com/Marak/Faker.js#readme
 */

export default {
  workspaceRole: Factory.extend({
    role() {
      let dice = Math.random();
      if (dice < 0.33) {
        return "OWNER";
      } else if (dice < 0.66) {
        return "DBA";
      } else {
        return "DEVELOPER";
      }
    },
    workspace: association(),

    user: association(),

    afterCreate(workspaceRole, server) {},
  }),
};
