export const OUTLOOK_OAUTH_CALLBACK_MESSAGE = "octomanger:outlook-oauth-callback";

export interface OutlookOAuthCallbackMessage {
  type: typeof OUTLOOK_OAUTH_CALLBACK_MESSAGE;
  code?: string;
  state?: string;
  error?: string;
  error_description?: string;
}
