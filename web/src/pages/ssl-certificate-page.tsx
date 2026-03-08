import { useEffect, useState } from "react";
import { AlertTriangle, CheckCircle, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

type CertMeta = {
  subject: string;
  issuer: string;
  not_before: string;
  not_after: string;
  sans: string[];
};

type SSLState = {
  cert: string;
  has_key: boolean;
  meta: CertMeta | null;
};

function isExpired(notAfter: string) {
  return new Date(notAfter) < new Date();
}

function isExpiringSoon(notAfter: string) {
  const delta = new Date(notAfter).getTime() - Date.now();
  return delta > 0 && delta < 30 * 24 * 60 * 60 * 1000; // 30 days
}

export function SSLCertificatePage() {
  const [state, setState] = useState<SSLState | null>(null);
  const [loading, setLoading] = useState(true);
  const [certInput, setCertInput] = useState("");
  const [keyInput, setKeyInput] = useState("");
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.getSSLCertificate();
      setState(data);
      setCertInput(data.cert ?? "");
      // Never pre-fill the key field
    } catch {
      setState({ cert: "", has_key: false, meta: null });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const handleSave = async () => {
    if (!certInput.trim()) {
      toast.error("请粘贴证书内容");
      return;
    }
    if (!keyInput.trim()) {
      toast.error("请粘贴私钥内容");
      return;
    }
    setSaving(true);
    try {
      const result = await api.setSSLCertificate({ cert: certInput.trim(), key: keyInput.trim() });
      setState({ cert: certInput.trim(), has_key: true, meta: result.meta });
      setKeyInput(""); // clear sensitive field after save
      toast.success("SSL 证书已保存");
    } catch (e) {
      toast.error(extractErrorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteSSLCertificate();
      setState({ cert: "", has_key: false, meta: null });
      setCertInput("");
      setKeyInput("");
      toast.success("SSL 证书已清除");
    } catch (e) {
      toast.error(extractErrorMessage(e));
    } finally {
      setDeleting(false);
    }
  };

  const meta = state?.meta ?? null;
  const expired = meta ? isExpired(meta.not_after) : false;
  const expiringSoon = meta ? isExpiringSoon(meta.not_after) : false;

  return (
    <div className="space-y-4">
      <PageHeader
        title="SSL 证书"
        description="配置 HTTPS 所需的 SSL/TLS 证书与私钥。"
        action={
          state?.meta && (
            <Button variant="destructive" size="sm" onClick={() => void handleDelete()} disabled={deleting}>
              <Trash2 className="mr-2 h-4 w-4" />
              {deleting ? "清除中..." : "清除证书"}
            </Button>
          )
        }
      />

      <div className="grid gap-4 xl:grid-cols-2">
        {/* Current cert info */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-primary" />
              当前证书信息
            </CardTitle>
            <CardDescription>已存储的证书解析结果</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">加载中...</p>
            ) : !meta ? (
              <p className="text-sm text-muted-foreground">暂未配置 SSL 证书。</p>
            ) : (
              <div className="space-y-3 text-sm">
                <div className="flex items-start justify-between gap-2 rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="shrink-0 text-muted-foreground">状态</span>
                  {expired ? (
                    <Badge variant="destructive" className="flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      已过期
                    </Badge>
                  ) : expiringSoon ? (
                    <Badge variant="secondary" className="flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      即将过期
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="flex items-center gap-1">
                      <CheckCircle className="h-3 w-3 text-green-500" />
                      有效
                    </Badge>
                  )}
                </div>
                <div className="flex items-start justify-between gap-2 rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="shrink-0 text-muted-foreground">主体</span>
                  <span className="break-all text-right font-mono text-xs">{meta.subject}</span>
                </div>
                <div className="flex items-start justify-between gap-2 rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="shrink-0 text-muted-foreground">颁发者</span>
                  <span className="break-all text-right font-mono text-xs">{meta.issuer}</span>
                </div>
                <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="text-muted-foreground">生效时间</span>
                  <span className="font-mono text-xs">{formatDateTime(meta.not_before)}</span>
                </div>
                <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="text-muted-foreground">到期时间</span>
                  <span className={`font-mono text-xs ${expired ? "text-destructive" : expiringSoon ? "text-yellow-500" : ""}`}>
                    {formatDateTime(meta.not_after)}
                  </span>
                </div>
                {meta.sans.length > 0 && (
                  <div className="rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                    <p className="mb-1 text-muted-foreground">SAN 域名</p>
                    <div className="flex flex-wrap gap-1 pt-1">
                      {meta.sans.map((san) => (
                        <Badge key={san} variant="secondary" className="font-mono text-xs">
                          {san}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
                <div className="flex items-center justify-between rounded-lg border border-border/80 bg-muted/30 px-3 py-2">
                  <span className="text-muted-foreground">私钥</span>
                  <Badge variant={state?.has_key ? "outline" : "secondary"}>{state?.has_key ? "已配置" : "未配置"}</Badge>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Upload / paste form */}
        <Card>
          <CardHeader>
            <CardTitle>粘贴新证书</CardTitle>
            <CardDescription>将 PEM 格式的证书和私钥粘贴到下方，然后保存。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="cert-input">证书（Certificate PEM）</Label>
              <Textarea
                id="cert-input"
                value={certInput}
                onChange={(e) => setCertInput(e.target.value)}
                placeholder={"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}
                className="h-40 font-mono text-xs"
                spellCheck={false}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="key-input">私钥（Private Key PEM）</Label>
              <Textarea
                id="key-input"
                value={keyInput}
                onChange={(e) => setKeyInput(e.target.value)}
                placeholder={"-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"}
                className="h-40 font-mono text-xs"
                spellCheck={false}
              />
              <p className="text-xs text-muted-foreground">私钥保存后不会再显示。如需更新请重新粘贴。</p>
            </div>
            <Button
              className="w-full"
              onClick={() => void handleSave()}
              disabled={saving || !certInput.trim() || !keyInput.trim()}
            >
              {saving ? "保存中..." : "保存证书"}
            </Button>
          </CardContent>
        </Card>

        {/* Raw PEM display */}
        {state?.cert && (
          <Card className="xl:col-span-2">
            <CardHeader>
              <CardTitle>证书内容（PEM）</CardTitle>
              <CardDescription>当前存储的证书原始内容，可直接复制。</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="overflow-auto rounded-lg border border-border/80 bg-muted/25 p-3 font-mono text-xs leading-5 whitespace-pre-wrap break-all">
                {state.cert}
              </pre>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
