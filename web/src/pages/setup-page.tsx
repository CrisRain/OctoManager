import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "@/lib/api";
import { setAdminKey } from "@/lib/auth";
import { extractErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type Step = "migrate" | "create-key" | "done";

export function SetupPage() {
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>("migrate");
  const [migrating, setMigrating] = useState(false);
  const [migrateError, setMigrateError] = useState<string | null>(null);
  const [keyName, setKeyName] = useState("Admin Key");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [rawKey, setRawKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  async function handleMigrate() {
    setMigrating(true);
    setMigrateError(null);
    try {
      await api.runMigration();
      setStep("create-key");
    } catch (e) {
      setMigrateError(extractErrorMessage(e));
    } finally {
      setMigrating(false);
    }
  }

  async function handleCreateKey() {
    setCreating(true);
    setCreateError(null);
    try {
      const result = await api.setup({ admin_key_name: keyName });
      setRawKey(result.raw_key);
      setAdminKey(result.raw_key);
      setStep("done");
    } catch (e) {
      setCreateError(extractErrorMessage(e));
    } finally {
      setCreating(false);
    }
  }

  async function handleCopy() {
    if (!rawKey) return;
    await navigator.clipboard.writeText(rawKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>OctoManger Setup</CardTitle>
          <CardDescription>
            {step === "migrate" && "运行数据库迁移以初始化 Schema。"}
            {step === "create-key" && "创建第一个 Admin API Key。"}
            {step === "done" && "初始化完成，请保存 Admin API Key。"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {step === "migrate" && (
            <>
              <p className="text-sm text-muted-foreground">
                将运行 GORM AutoMigrate 创建所有必要的数据表并初始化系统配置。对已有数据库安全。
              </p>
              {migrateError && (
                <p className="text-sm text-destructive">{migrateError}</p>
              )}
              <Button onClick={handleMigrate} disabled={migrating} className="w-full">
                {migrating ? "迁移中..." : "运行迁移"}
              </Button>
            </>
          )}

          {step === "create-key" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="key-name">Admin Key 名称</Label>
                <Input
                  id="key-name"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="Admin Key"
                />
              </div>
              {createError && (
                <p className="text-sm text-destructive">{createError}</p>
              )}
              <Button onClick={handleCreateKey} disabled={creating || !keyName.trim()} className="w-full">
                {creating ? "创建中..." : "创建 Admin Key"}
              </Button>
            </>
          )}

          {step === "done" && rawKey && (
            <>
              <p className="text-sm text-muted-foreground">
                请立即复制此密钥——关闭后将不再显示。已自动保存到当前浏览器会话。
              </p>
              <div className="rounded-md bg-muted px-3 py-2 font-mono text-sm break-all select-all">
                {rawKey}
              </div>
              <Button variant="outline" onClick={handleCopy} className="w-full">
                {copied ? "已复制！" : "复制到剪贴板"}
              </Button>
              <Button onClick={() => navigate("/dashboard")} className="w-full">
                前往仪表板
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
