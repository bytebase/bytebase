import { Factory } from "miragejs";

export default {
  dataSource: Factory.extend({
    name(i) {
      return "ds" + i;
    },
    type(i) {
      return i == 0 ? "ADMIN" : "NORMAL";
    },
    username() {
      return "root";
    },
    password(i) {
      return "pwd" + i;
    },
  }),
};
