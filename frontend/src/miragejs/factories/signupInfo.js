import { Factory } from "miragejs";

export default {
  signupInfo: Factory.extend({
    email() {
      return "foo@example.com";
    },
    password() {
      return "blablabla";
    },
    name() {
      return "foo";
    },
  }),
};
