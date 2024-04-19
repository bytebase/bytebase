import { v4 as uuidv4 } from "uuid";
import {
  DataSource,
  DataSourceType,
  DataSource_AuthenticationType,
} from "../proto/v1/instance_service";

export const emptyDataSource = () => {
  return DataSource.fromPartial({
    type: DataSourceType.ADMIN,
    id: uuidv4(),
    authenticationType: DataSource_AuthenticationType.PASSWORD,
  });
};

export const unknownDataSource = () => {
  return {
    ...emptyDataSource(),
    title: "<<Unknown data source>>",
  };
};
