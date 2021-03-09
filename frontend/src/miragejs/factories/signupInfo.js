import { Factory } from "miragejs";

export default {
  signupInfo: Factory.extend({
    usename() {
      return "foo@example.com";
    },
    password() {
      return "blablabla";
    },
  }),
};
