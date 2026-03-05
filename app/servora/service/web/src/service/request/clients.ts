import {
  createAuthServiceClient,
  createUserServiceClient,
  type AuthService,
  type UserService,
} from "../gen/servora/service/v1";

import {
  createRequestHandler,
  type RequestHandlerOptions,
} from "./requestHandler";

export type servoraClients = {
  auth: AuthService;
  user: UserService;
};

export function createservoraClients(
  options: RequestHandlerOptions = {},
): servoraClients {
  const handler = createRequestHandler(options);
  return {
    auth: createAuthServiceClient(handler),
    user: createUserServiceClient(handler),
  };
}

export * from "../gen/servora/service/v1";
