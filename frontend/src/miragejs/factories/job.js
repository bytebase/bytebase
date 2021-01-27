import { Factory } from "miragejs";
import faker from "faker";

export default {
  job: Factory.extend({
    name() {
      return "pipeline job " + faker.fake("{{lorem.word}}");
    },
    slug(i) {
      return 1000 + i;
    },
    lastUpdated() {
      return Date.now() - 3600 * 1000;
    },
    status() {
      let dice = Math.random();
      return dice < 0.33 ? "RUNNING" : dice < 0.66 ? "SUCCEEDED" : "FAILED";
    },
    afterCreate(job, server) {
      let stepList = [];
      for (let i = 0; i < 3; i++) {
        stepList.push(
          server.create("step", {
            job,
          })
        );
      }
      job.update({ step: stepList });
    },
  }),
};
