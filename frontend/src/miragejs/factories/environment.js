import { Factory } from "miragejs";

export default {
  environment: Factory.extend({
    name(i) {
      if (i == 0) {
        return "Sandbox A";
      } else if (i == 1) {
        return "Integration";
      } else if (i == 2) {
        return "Staging";
      } else {
        if (i == 3) {
          return "Prod";
        }
        return "Prod " + (i - 2);
      }
    },
    order(i) {
      return i;
    },
    host(i) {
      if (i == 0) {
        return "localhost";
      } else if (i == 1) {
        return "127.0.0.1";
      } else if (i == 2) {
        return "13.24.32.122";
      } else {
        return "mydb.com";
      }
    },
    port(i) {
      if (i == 0) {
        return "3306";
      } else if (i == 1) {
        return "";
      } else if (i == 2) {
        return "15202";
      } else {
        return "5432";
      }
    },
  }),
};
