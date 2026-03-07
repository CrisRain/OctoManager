import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { api } from "@/lib/api";

export interface OutlookOAuthConfig {
  clientId: string;
  clientSecret: string;
  tenant: string;
  redirectUri: string;
  scope: string;
  mailbox: string;
  graphBaseURL: string;
}

export const DEFAULT_OUTLOOK_CONFIG: OutlookOAuthConfig = {
  clientId: "",
  clientSecret: "",
  tenant: "consumers",
  redirectUri: "http://localhost:8080/oauth/callback",
  scope: "offline_access openid profile email https://graph.microsoft.com/Mail.Read",
  mailbox: "INBOX",
  graphBaseURL: "https://graph.microsoft.com/v1.0",
};

const CONFIG_KEY = "outlook_oauth_config";

export function useOutlookConfig() {
  const [config, setConfig] = useState<OutlookOAuthConfig>(DEFAULT_OUTLOOK_CONFIG);
  const [configLoading, setConfigLoading] = useState(true);

  useEffect(() => {
    api
      .getConfig(CONFIG_KEY)
      .then((res) => {
        if (res.value && typeof res.value === "object") {
          setConfig({ ...DEFAULT_OUTLOOK_CONFIG, ...(res.value as Partial<OutlookOAuthConfig>) });
        }
      })
      .catch(() => {
        // Not found or error — keep defaults
      })
      .finally(() => setConfigLoading(false));
  }, []);

  const saveConfig = async (next: OutlookOAuthConfig): Promise<void> => {
    await api.setConfig(CONFIG_KEY, next);
    setConfig(next);
  };

  return { config, configLoading, saveConfig };
}

function splitScopes(raw: string): string[] {
  const normalized = raw.trim().replace(/,/g, " ");
  if (!normalized) return [];
  return Array.from(new Set(normalized.split(/\s+/).map((s) => s.trim()).filter(Boolean)));
}

interface OutlookConfigPanelProps {
  config: OutlookOAuthConfig;
  configLoading: boolean;
  onSave: (config: OutlookOAuthConfig) => Promise<void>;
}

export function OutlookConfigPanel({ config, configLoading, onSave }: OutlookConfigPanelProps) {
  const [draft, setDraft] = useState<OutlookOAuthConfig>(config);
  const [saving, setSaving] = useState(false);

  // Sync draft when config loads from backend
  useEffect(() => {
    setDraft(config);
  }, [config]);

  const scopeCount = splitScopes(draft.scope).length;

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(draft);
      toast.success("Outlook OAuth 配置已保存");
    } catch {
      toast.error("保存配置失败");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setDraft(DEFAULT_OUTLOOK_CONFIG);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Outlook OAuth Config</CardTitle>
        <CardDescription>
          手动添加和批量导入共用的 OAuth 配置。批量导入仅使用 <code>client_id</code>，<code>client_secret</code> 仅用于手动 OAuth 流程。
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="cfg-client-id">Client ID</Label>
              <Input
                id="cfg-client-id"
                value={draft.clientId}
                onChange={(e) => setDraft((prev) => ({ ...prev, clientId: e.target.value }))}
                placeholder="Azure app client_id"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cfg-client-secret">Client Secret</Label>
              <Input
                id="cfg-client-secret"
                type="password"
                value={draft.clientSecret}
                onChange={(e) => setDraft((prev) => ({ ...prev, clientSecret: e.target.value }))}
                placeholder="Optional for public client"
              />
              <p className="text-xs text-muted-foreground">Used by manual OAuth flows only. Batch Import does not send it.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="cfg-tenant">Tenant</Label>
              <Input
                id="cfg-tenant"
                value={draft.tenant}
                onChange={(e) => setDraft((prev) => ({ ...prev, tenant: e.target.value }))}
                placeholder="consumers / common / tenant-id"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cfg-redirect-uri">Redirect URI</Label>
              <Input
                id="cfg-redirect-uri"
                value={draft.redirectUri}
                onChange={(e) => setDraft((prev) => ({ ...prev, redirectUri: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cfg-mailbox">Mailbox</Label>
              <Input
                id="cfg-mailbox"
                value={draft.mailbox}
                onChange={(e) => setDraft((prev) => ({ ...prev, mailbox: e.target.value }))}
                placeholder="INBOX"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cfg-graph-base-url">Graph Base URL</Label>
              <Input
                id="cfg-graph-base-url"
                value={draft.graphBaseURL}
                onChange={(e) => setDraft((prev) => ({ ...prev, graphBaseURL: e.target.value }))}
                placeholder="https://graph.microsoft.com/v1.0"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="cfg-scope">OAuth Scope</Label>
            <Textarea
              id="cfg-scope"
              className="min-h-[90px] font-mono text-xs"
              value={draft.scope}
              onChange={(e) => setDraft((prev) => ({ ...prev, scope: e.target.value }))}
            />
            <p className="text-xs text-muted-foreground">已激活 scope：{scopeCount} 个</p>
          </div>

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={handleReset} disabled={saving || configLoading}>
              重置为默认值
            </Button>
            <Button type="button" onClick={() => void handleSave()} disabled={saving || configLoading}>
              {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              保存配置
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
