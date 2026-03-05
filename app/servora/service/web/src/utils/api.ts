import {
  createservoraClients,
  type authservicev1_LoginByEmailPasswordResponse,
  type authservicev1_SignupByEmailResponse,
  type userservicev1_CurrentUserInfoResponse,
} from "@/service/request/clients";

const publicClients = createservoraClients();

export const api = {
  login(
    email: string,
    password: string,
  ): Promise<authservicev1_LoginByEmailPasswordResponse> {
    return publicClients.auth.LoginByEmailPassword({ email, password });
  },

  signup(
    name: string,
    email: string,
    password: string,
    passwordConfirm: string,
  ): Promise<authservicev1_SignupByEmailResponse> {
    return publicClients.auth.SignupByEmail({
      name,
      email,
      password,
      passwordConfirm,
    });
  },

  getCurrentUser(
    token: string,
  ): Promise<userservicev1_CurrentUserInfoResponse> {
    const clients = createservoraClients({ getAccessToken: () => token });
    return clients.user.CurrentUserInfo({});
  },
};
