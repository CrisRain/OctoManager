import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { LogOut, TerminalSquare } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, extractErrorMessage, fetchHealth } from "@/lib/api";
import { clearAdminKey, getAdminKey } from "@/lib/auth";
import { formatDateTime } from "@/lib/format";

const KNOWN_CONFIGS: { key: string; label: string; description: string }[] = [
  { key: "app.name", label: "应用名称", description: "平台显示名称" },
  { key: "job.default_timeout_minutes", label: "任务超时（分钟）", description: "默认任务执行超时" },
  { key: "job.max_concurrency", label: "最大并发数", description: "Worker 最大并发任务数" },
];

export function SettingsPage() {
  const navigate = useNavigate();
  const [health, setHealth] = useState<{ status: string; time: string } | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);
  const [configs, setConfigs] = useState<Record<string, string>>({});
  const [configEdits, setConfigEdits] = useState<Record<string, string>>({});
  const [configSaving, setConfigSaving] = useState<Record<string, boolean>>({});

  const checkHealth = async () => {
    try {
      setHealthLoading(true);
      const result = await fetchHealth();
      setHealth(result);
      toast.success("健康检查成功");
    } catch (error) {
      const message = error instanceof Error ? error.message : "健康检查失败";
      toast.error(message);
    } finally {
      setHealthLoading(false);
    }
  };

  const loadConfigs = async () => {
    const values: Record<string, string> = {};
    for (const c of KNOWN_CONFIGS) {
      try {
        const res = await api.getConfig(c.key);
        values[c.key] = JSON.stringify(res.value ?? "");
      } catch {
        values[c.key] = "";
      }
    }
    setConfigs(values);
    setConfigEdits(values);
  };

  useEffect(() => {
    void checkHealth();
    void loadConfigs();
  }, []);

  const handleSaveConfig = async (key: string) => {
    const raw = configEdits[key] ?? "";
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      toast.error(`${key}: 值必须是有效 JSON`);
      return;
    }
    setConfigSaving((prev) => ({ ...prev, [key]: true }));
    try {
      await api.setConfig(key, parsed);
      setConfigs((prev) => ({ ...prev, [key]: raw }));
      toast.success("已保存");
    } catch (e) {
      toast.error(extractErrorMessage(e));
    } finally {
      setConfigSaving((prev) => ({ ...prev, [key]: false }));
    }
  };

  const handleSignOut = () => {
    clearAdminKey();
    navigate("/auth", { replace: true });
  };

  const adminKey = getAdminKey();
  const apiBase = (import.meta.env.VITE_API_BASE as string | undefined) || "(same-origin)";

  return (
    <div className="space-y-4">
      <PageHeader
        title="设置"
        description="环境信息、健康检查与系统配置。"
        action={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => void checkHealth()} disabled={healthLoading}>
              {healthLoading ? "检查中..." : "重新检查健康状态"}
            </Button>
            <Button variant="outline" onClick={handleSignOut}>
              <LogOut className="mr-2 h-4 w-4" />
              退出登录
            </Button>
          </div>
        }
      />

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>运行时</CardTitle>
            <CardDescription>前端运行时配置快照</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
              <span className="text-muted-foreground">VITE_API_BASE</span>
              <code className="font-mono text-xs">{apiBase}</code>
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
              <span className="text-muted-foreground">健康状态</span>
              <Badge variant={health?.status === "ok" ? "outline" : "secondary"}>{health?.status ?? "unknown"}</Badge>
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
              <span className="text-muted-foreground">健康检查时间</span>
              <span>{health ? formatDateTime(health.time) : "-"}</span>
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
              <span className="text-muted-foreground">Admin Key 前缀</span>
              <code className="font-mono text-xs">{adminKey ? adminKey.slice(0, 8) + "..." : "未设置"}</code>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>系统配置</CardTitle>
            <CardDescription>存储在数据库中的系统级配置项。值为 JSON 格式。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {KNOWN_CONFIGS.map((c) => (
              <div key={c.key} className="space-y-1">
                <Label htmlFor={`cfg-${c.key}`} className="text-sm">
                  {c.label}
                  <span className="ml-1 text-xs text-muted-foreground">({c.key})</span>
                </Label>
                <p className="text-xs text-muted-foreground">{c.description}</p>
                <div className="flex gap-2">
                  <Input
                    id={`cfg-${c.key}`}
                    value={configEdits[c.key] ?? ""}
                    onChange={(e) => setConfigEdits((prev) => ({ ...prev, [c.key]: e.target.value }))}
                    placeholder='例如: "OctoManger" 或 30'
                    className="font-mono text-sm"
                  />
                  <Button
                    size="sm"
                    disabled={configSaving[c.key] || configEdits[c.key] === configs[c.key]}
                    onClick={() => void handleSaveConfig(c.key)}
                  >
                    {configSaving[c.key] ? "保存中..." : "保存"}
                  </Button>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="xl:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TerminalSquare className="h-5 w-5 text-primary" />
              快速调用
            </CardTitle>
            <CardDescription>用于联调的最小 curl 示例。</CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="overflow-auto rounded-lg border border-border/80 bg-muted/25 p-3 font-mono text-xs leading-5">
{`curl -H "X-Api-Key: <your-admin-key>" http://localhost:8080/api/v1/account-types/
curl -H "X-Api-Key: <your-admin-key>" http://localhost:8080/api/v1/accounts/
curl -H "X-Api-Key: <your-admin-key>" http://localhost:8080/api/v1/email/accounts/
curl -H "X-Api-Key: <your-admin-key>" http://localhost:8080/api/v1/jobs/
curl -H "X-Api-Key: <your-admin-key>" http://localhost:8080/api/v1/octo-modules/

# Webhook via API key (webhook role):
curl -X POST -H "X-Api-Key: <webhook-key>" http://localhost:8080/webhooks/<slug>

# System setup (no auth required):
curl http://localhost:8080/api/v1/system/status
curl -X POST http://localhost:8080/api/v1/system/setup`}
            </pre>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
