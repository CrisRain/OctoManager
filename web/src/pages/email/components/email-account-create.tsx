import { useMemo, useState } from "react";
import { Loader2, Settings2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { parseJSONObjectText } from "@/lib/format";
import type { JsonObject } from "@/types";
import type { OutlookOAuthConfig } from "./outlook-config";
import {
  OUTLOOK_OAUTH_CALLBACK_MESSAGE,
  type OutlookOAuthCallbackMessage,
} from "./outlook-oauth-bridge";

interface EmailAccountCreateProps {
  config: OutlookOAuthConfig;
  onSuccess: () => void;
}

const CALLBACK_TIMEOUT_MS = 3 * 60 * 1000;

function splitScopes(raw: string): string[] {
  const normalized = raw.trim().replace(/,/g, " ");
  if (!normalized) {
    return [];
  }
  const scopes = normalized
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean);
  return Array.from(new Set(scopes));
}

function toBase64URL(input: Uint8Array): string {
  let binary = "";
  for (const item of input) {
    binary += String.fromCharCode(item);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function randomBase64URL(size = 32): string {
  const bytes = new Uint8Array(size);
  crypto.getRandomValues(bytes);
  return toBase64URL(bytes);
}

async function createPkcePair() {
  const verifier = randomBase64URL(48);
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return {
    verifier,
    challenge: toBase64URL(new Uint8Array(digest)),
  };
}

function isOutlookConsumerAddress(address: string) {
  const [, domain = ""] = address.toLowerCase().split("@");
  return ["outlook.com", "hotmail.com", "live.com", "msn.com"].includes(domain);
}

function resolveTenant(address: string, configuredTenant: string) {
  const trimmed = configuredTenant.trim();
  if (trimmed) {
    return trimmed;
  }
  return isOutlookConsumerAddress(address) ? "consumers" : "common";
}

function buildTokenURL(tenant: string) {
  return `https://login.microsoftonline.com/${encodeURIComponent(tenant)}/oauth2/v2.0/token`;
}

function waitForOAuthCallback(popup: Window, expectedState: string): Promise<string> {
  return new Promise((resolve, reject) => {
    let done = false;

    const cleanup = () => {
      window.removeEventListener("message", handleMessage);
      window.clearTimeout(timeoutID);
      window.clearInterval(closeCheckID);
    };

    const finish = (next: () => void) => {
      if (done) {
        return;
      }
      done = true;
      cleanup();
      next();
    };

    const handleMessage = (event: MessageEvent<OutlookOAuthCallbackMessage>) => {
      if (event.origin !== window.location.origin) {
        return;
      }

      const payload = event.data;
      if (!payload || payload.type !== OUTLOOK_OAUTH_CALLBACK_MESSAGE) {
        return;
      }

      if ((payload.state ?? "") !== expectedState) {
        finish(() => reject(new Error("OAuth state 不匹配")));
        return;
      }

      if (payload.error) {
        const detail = payload.error_description?.trim() || payload.error;
        finish(() => reject(new Error(`OAuth 失败：${detail}`)));
        return;
      }

      if (!payload.code?.trim()) {
        finish(() => reject(new Error("OAuth 回调缺少 code")));
        return;
      }

      finish(() => resolve(payload.code!.trim()));
    };

    const timeoutID = window.setTimeout(() => {
      finish(() => reject(new Error("OAuth 回调超时")));
    }, CALLBACK_TIMEOUT_MS);

    const closeCheckID = window.setInterval(() => {
      if (!popup.closed) {
        return;
      }
      finish(() => reject(new Error("OAuth 窗口在回调前已关闭")));
    }, 500);

    window.addEventListener("message", handleMessage);
  });
}

function openOAuthPopup(url: string): Window {
  const width = 520;
  const height = 760;
  const left = window.screenX + Math.max(0, (window.outerWidth - width) / 2);
  const top = window.screenY + Math.max(0, (window.outerHeight - height) / 2);
  const features = [
    `width=${Math.round(width)}`,
    `height=${Math.round(height)}`,
    `left=${Math.round(left)}`,
    `top=${Math.round(top)}`,
    "resizable=yes",
    "scrollbars=yes",
  ].join(",");

  const popup = window.open(url, "outlook-oauth", features);
  if (!popup) {
    throw new Error("弹窗被拦截，请允许弹出窗口后重试。");
  }
  popup.focus();
  return popup;
}

export function EmailAccountCreate({ config, onSuccess }: EmailAccountCreateProps) {
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    address: "",
    status: "0",
    loginHint: "",
  });

  const [graphConfigDialogOpen, setGraphConfigDialogOpen] = useState(false);
  const [graphConfigText, setGraphConfigText] = useState("{}");
  const [graphConfigDraft, setGraphConfigDraft] = useState("{}");

  const graphOverrideCount = useMemo(() => {
    try {
      return Object.keys(parseJSONObjectText(graphConfigText, "graph_config")).length;
    } catch {
      return 0;
    }
  }, [graphConfigText]);

  const openGraphConfigEditor = () => {
    setGraphConfigDraft(graphConfigText);
    setGraphConfigDialogOpen(true);
  };

  const saveGraphConfigEditor = () => {
    try {
      const parsed = parseJSONObjectText(graphConfigDraft, "graph_config");
      setGraphConfigText(JSON.stringify(parsed, null, 2));
      setGraphConfigDialogOpen(false);
      toast.success("graph_config 覆写已更新");
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const address = form.address.trim().toLowerCase();
    const scopes = splitScopes(config.scope);
    const clientId = config.clientId.trim();
    const clientSecret = config.clientSecret.trim();
    const redirectURI = config.redirectUri.trim();
    const tenant = resolveTenant(address, config.tenant);
    const loginHint = form.loginHint.trim();
    const mailbox = config.mailbox.trim() || "INBOX";
    const graphBaseURL = config.graphBaseURL.trim() || "https://graph.microsoft.com/v1.0";

    if (!address || !address.includes("@")) {
      toast.error("请输入有效的邮箱地址。");
      return;
    }
    if (!clientId) {
      toast.error("Client ID 为必填项，请在配置标签页中填写。");
      return;
    }
    if (!redirectURI) {
      toast.error("Redirect URI 为必填项，请在配置标签页中填写。");
      return;
    }
    if (scopes.length === 0) {
      toast.error("请在配置标签页中配置至少一个 scope。");
      return;
    }

    let redirectOrigin = "";
    try {
      redirectOrigin = new URL(redirectURI).origin;
    } catch {
      toast.error("Redirect URI 格式无效。");
      return;
    }
    if (redirectOrigin !== window.location.origin) {
      toast.error("Redirect URI 必须与当前页面同源，以支持自动回调处理。");
      return;
    }

    let graphConfigOverrides: JsonObject;
    try {
      graphConfigOverrides = parseJSONObjectText(graphConfigText, "graph_config");
    } catch (error) {
      toast.error(extractErrorMessage(error));
      return;
    }

    setLoading(true);
    let popup: Window | null = null;
    try {
      const state = randomBase64URL(16);
      const pkce = await createPkcePair();

      const authorize = await api.buildOutlookAuthorizeURL({
        client_id: clientId,
        tenant,
        redirect_uri: redirectURI,
        scope: scopes,
        state,
        login_hint: loginHint || undefined,
        code_challenge: pkce.challenge,
        code_challenge_method: "S256",
      });

      popup = openOAuthPopup(authorize.authorize_url);
      const expectedState = authorize.state?.trim() || state;
      const authCode = await waitForOAuthCallback(popup, expectedState);

      // Send client_secret when configured (confidential client) together with code_verifier (PKCE).
      // Public clients omit client_secret; confidential clients in Azure AD require both.
      const token = await api.exchangeOutlookCode({
        client_id: clientId,
        tenant,
        redirect_uri: redirectURI,
        code: authCode,
        scope: scopes,
        code_verifier: pkce.verifier,
        ...(clientSecret ? { client_secret: clientSecret } : {}),
      });

      const refreshToken = token.refresh_token?.trim();
      if (!refreshToken) {
        throw new Error("Token 换取成功，但 refresh_token 为空，请确认已开启 offline_access scope。");
      }

      const remoteScopes = splitScopes(token.scope ?? "");
      const resolvedScopes = remoteScopes.length > 0 ? remoteScopes : scopes;

      const graphConfig = {
        auth_method: "graph_oauth2",
        username: address,
        client_id: clientId,
        refresh_token: refreshToken,
        tenant,
        scope: resolvedScopes,
        token_url: token.token_url?.trim() || buildTokenURL(tenant),
        graph_base_url: graphBaseURL,
        mailbox,
        ...(clientSecret ? { client_secret: clientSecret } : {}),
        ...(token.access_token?.trim() ? { access_token: token.access_token.trim() } : {}),
        ...(token.expires_at?.trim() ? { token_expires_at: token.expires_at.trim() } : {}),
      };

      await api.createEmailAccount({
        address,
        provider: "outlook",
        status: Number(form.status),
        graph_config: {
          ...graphConfig,
          ...graphConfigOverrides,
        },
      });

      toast.success(`Outlook 账号 ${address} 已添加`);
      setForm((prev) => ({
        ...prev,
        address: "",
        loginHint: "",
      }));
      onSuccess();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      if (popup && !popup.closed) {
        popup.close();
      }
      setLoading(false);
    }
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>手动添加 Outlook 账号</CardTitle>
          <CardDescription>
            点击「添加账号」后将弹出 Outlook OAuth 授权窗口，回调到
            {" "}
            <code className="rounded bg-muted px-1 py-0.5 text-xs font-mono">/oauth/callback</code>
            {" "}后自动完成账号创建。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="email-address">邮箱地址</Label>
                <Input
                  id="email-address"
                  value={form.address}
                  onChange={(e) => setForm((prev) => ({ ...prev, address: e.target.value }))}
                  placeholder="user@outlook.com"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="email-status">状态</Label>
                <Select
                  value={form.status}
                  onValueChange={(value) => setForm((prev) => ({ ...prev, status: value }))}
                >
                  <SelectTrigger id="email-status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="0">待验证 (0)</SelectItem>
                    <SelectItem value="1">已验证 (1)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="oauth-login-hint">登录提示</Label>
                <Input
                  id="oauth-login-hint"
                  value={form.loginHint}
                  onChange={(e) => setForm((prev) => ({ ...prev, loginHint: e.target.value }))}
                  placeholder="可选，如 user@outlook.com"
                />
              </div>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div className="text-xs text-muted-foreground">
                graph_config 覆写：{graphOverrideCount} 项
              </div>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={openGraphConfigEditor}
                  disabled={loading}
                >
                  <Settings2 className="mr-2 h-4 w-4" />
                  编辑 graph_config
                </Button>
                <Button type="submit" disabled={loading}>
                  {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                  添加账号
                </Button>
              </div>
            </div>
          </form>
        </CardContent>
      </Card>

      <Dialog open={graphConfigDialogOpen} onOpenChange={setGraphConfigDialogOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>编辑 graph_config 覆写</DialogTitle>
            <DialogDescription>
              OAuth 令牌换取后合并的高级覆写项，使用 JSON 对象格式。
            </DialogDescription>
          </DialogHeader>
          <Textarea
            className="min-h-[320px] font-mono text-xs"
            value={graphConfigDraft}
            onChange={(e) => setGraphConfigDraft(e.target.value)}
          />
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setGraphConfigDialogOpen(false)}
            >
              取消
            </Button>
            <Button type="button" onClick={saveGraphConfigEditor}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
