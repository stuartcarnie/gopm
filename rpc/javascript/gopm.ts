import {Empty} from "google-protobuf/google/protobuf/empty_pb";
import {GopmPromiseClient} from "./service_grpc_web_pb";

export {Empty, GopmPromiseClient};
export * from "./service_pb"

export const client = new GopmPromiseClient("http://localhost:9001");

