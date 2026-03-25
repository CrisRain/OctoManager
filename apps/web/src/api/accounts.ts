import { client } from "@/shared/api/generated/client";
import type {
  Account,
  AccountExecuteInput,
  AccountExecuteResult,
  AccountCreateInput,
  AccountPatchInput,
  ListAccountsResponse,
} from "@/types";

export const listAccounts = (): Promise<ListAccountsResponse> => client.listAccounts();

export const getAccount = (id: number): Promise<Account> =>
  client.getAccount({ path: { id } });

export const createAccount = (payload: AccountCreateInput) =>
  client.createAccount({ body: payload });

export const patchAccount = (id: number, payload: AccountPatchInput) =>
  client.patchAccount({ path: { id }, body: payload });

export const executeAccount = (
  id: number,
  payload: AccountExecuteInput,
): Promise<AccountExecuteResult> =>
  client.executeAccount({ path: { id }, body: payload });

export const deleteAccount = (id: number) =>
  client.deleteAccount({ path: { id } });
