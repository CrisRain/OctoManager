import { client } from "@/shared/api/generated/client";
import type { AccountType, AccountTypeCreateInput, AccountTypePatchInput, ListAccountTypesResponse } from "@/types";

export const listAccountTypes = (): Promise<ListAccountTypesResponse> =>
  client.listAccountTypes();

export const getAccountType = (key: string): Promise<AccountType> =>
  client.getAccountType({ path: { key } });

export const createAccountType = (payload: AccountTypeCreateInput) =>
  client.createAccountType({ body: payload });

export const patchAccountType = (key: string, payload: AccountTypePatchInput): Promise<AccountType> =>
  client.patchAccountType({ path: { key }, body: payload });

export const deleteAccountType = (key: string) =>
  client.deleteAccountType({ path: { key } });
