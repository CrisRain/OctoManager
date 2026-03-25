import { client } from "@/shared/api/generated/client";
import type {
  EmailAccount,
  EmailAccountCreateInput,
  EmailAccountPatchInput,
  EmailBulkImportInput,
  EmailBulkImportResult,
  EmailLatestMessageResult,
  EmailMailboxListResult,
  EmailMessageDetail,
  EmailMessageListResult,
  EmailPreviewInput,
  ListEmailAccountsResponse,
  OutlookAuthorizeURLResult,
  OutlookExchangeCodeInput,
} from "@/types";

export const listEmailAccounts = (): Promise<ListEmailAccountsResponse> =>
  client.listEmailAccounts();

export const getEmailAccount = (id: number): Promise<EmailAccount> =>
  client.getEmailAccount({ path: { id } });

export const bulkImportEmailAccounts = (
  lines: EmailBulkImportInput["lines"],
): Promise<EmailBulkImportResult> => client.bulkImportEmailAccounts({ body: { lines } });

export const createEmailAccount = (payload: EmailAccountCreateInput) =>
  client.createEmailAccount({ body: payload });

export const patchEmailAccount = (id: number, payload: EmailAccountPatchInput) =>
  client.patchEmailAccount({ path: { id }, body: payload });

export const deleteEmailAccount = (id: number) =>
  client.deleteEmailAccount({ path: { id } });

export const buildOutlookAuthorizeURL = (id: number): Promise<OutlookAuthorizeURLResult> =>
  client.buildOutlookAuthorizeURL({ path: { id } });

export const exchangeOutlookCode = (id: number, payload: OutlookExchangeCodeInput) =>
  client.exchangeOutlookCode({ path: { id }, body: payload });

export const listEmailMailboxes = (
  id: number,
  pattern?: string,
): Promise<EmailMailboxListResult> =>
  client.listEmailMailboxes({
    path: { id },
    query: pattern ? { pattern } : undefined,
  });

export const listEmailMessages = (
  id: number,
  options: { mailbox?: string; limit?: number; offset?: number },
): Promise<EmailMessageListResult> =>
  client.listEmailMessages({
    path: { id },
    query: options,
  });

export const getEmailMessage = (
  id: number,
  messageId: string,
): Promise<EmailMessageDetail> =>
  client.getEmailMessage({ path: { id, message_id: messageId } });

export const getLatestEmailMessage = (
  id: number,
  mailbox?: string,
): Promise<EmailLatestMessageResult> =>
  client.getLatestEmailMessage({ path: { id }, query: mailbox ? { mailbox } : undefined });

export const previewEmailMailboxes = (
  payload: EmailPreviewInput,
): Promise<EmailMailboxListResult> => client.previewEmailMailboxes({ body: payload });

export const previewLatestEmailMessage = (
  payload: EmailPreviewInput,
): Promise<EmailLatestMessageResult> => client.previewLatestEmailMessage({ body: payload });
