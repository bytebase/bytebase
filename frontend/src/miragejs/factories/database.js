import { Factory } from "miragejs";
import faker from "faker";

export default {
  database: Factory.extend({
    name(i) {
      return "db" + i;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    syncStatus(i) {
      if (i % 3 == 0) {
        return "OK";
      }
      if (i % 3 == 1) {
        return "MISMATCH";
      }
      if (i % 3 == 2) {
        return "NOT_FOUND";
      }
    },
    fingerprint(i) {
      return faker.fake("{{random.alpha}}");
    },
  }),
};
