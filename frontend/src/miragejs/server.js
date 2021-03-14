import { Server, JSONAPISerializer } from "miragejs";
import factories from "./factories";
import fixtures from "./fixtures";
import routes from "./routes";
import models from "./models";
import seeds from "./seeds";

const config = (environment) => {
  const config = {
    environment,
    factories,
    models,
    routes,
    seeds,
    serializers: {
      application: JSONAPISerializer.extend({
        shouldIncludeLinkageData(relationshipKey, model) {
          return true;
        },
        keyForAttribute(modelName) {
          return modelName;
        },
        keyForModel(modelName) {
          return modelName;
        },
        keyForRelationship(modelName) {
          return modelName;
        },
        typeKeyForModel(model) {
          return model.modelName;
        },
      }),
    },
  };

  if (Object.keys(fixtures).length) {
    config.fixtures = fixtures;
  }

  return config;
};

export function makeServer({ environment = "development" } = {}) {
  const server = new Server(config(environment));
  console.log("===Mirage DB Dump Start===");
  console.log(server.db.dump());
  console.log("==========================");
  return server;
}
