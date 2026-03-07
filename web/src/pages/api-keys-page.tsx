import { useCallback, useEffect, useState } from "react";
import { Copy, Loader2, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { ApiKey, CreateApiKeyResult } from "@/types";

export function ApiKeysPage() {
  const [items, setItems] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createRole, setCreateRole] = useState<"admin" | "webhook">("admin");
  const [createScope, setCreateScope] = useState("*");
  const [createLoading, setCreateLoading] = useState(false);
  const [createdResult, setCreatedResult] = useState<CreateApiKeyResult | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listApiKeys();
      setItems(res);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleCreate = async () => {
    if (!createName.trim()) {
      toast.error("名称不能为空");
      return;
    }
    setCreateLoading(true);
    try {
      const res = await api.createApiKey({
        name: createName.trim(),
        role: createRole,
        webhook_scope: createRole === "webhook" ? (createScope.trim() || "*") : undefined,
      });
      setCreatedResult(res);
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setCreateLoading(false);
    }
  };

  const handleToggle = async (item: ApiKey, enabled: boolean) => {
    try {
      await api.setApiKeyEnabled(item.id, enabled);
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleDelete = async (item: ApiKey) => {
    if (!confirm(`确定要删除 API 密钥 "${item.name}" 吗？`)) return;
    try {
      await api.deleteApiKey(item.id);
      toast.success("API 密钥已删除");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleCopy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success("已复制到剪贴板");
    } catch {
      toast.error("复制失败");
    }
  };

  function openCreate() {
    setCreateOpen(true);
    setCreatedResult(null);
    setCreateName("");
    setCreateRole("admin");
    setCreateScope("*");
  }

  return (
    <div className="space-y-4">
      <PageHeader title="API 密钥" description="管理外部访问用的 API 密钥。">
        <Button onClick={openCreate}>
          <Plus className="mr-2 h-4 w-4" />
          创建密钥
        </Button>
      </PageHeader>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle>密钥列表</CardTitle>
            <CardDescription>共 {items.length} 条</CardDescription>
          </div>
          <Button variant="outline" size="icon" onClick={() => void load()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground">加载中...</div>
          ) : items.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无 API 密钥。
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>前缀</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead>Webhook 范围</TableHead>
                  <TableHead>启用</TableHead>
                  <TableHead>最近使用</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="font-mono text-xs">{item.key_prefix}</TableCell>
                    <TableCell>
                      <Badge variant={item.role === "admin" ? "default" : "secondary"}>
                        {item.role}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {item.role === "webhook" ? (item.webhook_scope ?? "*") : "-"}
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={item.enabled}
                        onCheckedChange={(checked) => void handleToggle(item, checked)}
                      />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {item.last_used_at ? formatDateTime(item.last_used_at) : "-"}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDateTime(item.created_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => void handleDelete(item)}
                        title="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>创建 API 密钥</DialogTitle>
            <DialogDescription>密钥仅在创建后显示一次，请妥善保存。</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label>名称</Label>
              <Input
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                placeholder="生产环境集成"
                disabled={!!createdResult}
              />
            </div>

            <div className="space-y-2">
              <Label>角色</Label>
              <Select
                value={createRole}
                onValueChange={(v) => setCreateRole(v as "admin" | "webhook")}
                disabled={!!createdResult}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">admin — 所有 API</SelectItem>
                  <SelectItem value="webhook">webhook — 仅触发器</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {createRole === "webhook" && (
              <div className="space-y-2">
                <Label>Webhook 范围</Label>
                <Input
                  value={createScope}
                  onChange={(e) => setCreateScope(e.target.value)}
                  placeholder="* 代表所有触发器，或填写具体 slug"
                  disabled={!!createdResult}
                />
                <p className="text-xs text-muted-foreground">
                  填 <code>*</code> 允许触发所有 Webhook，或填写单个触发器的 slug。
                </p>
              </div>
            )}

            {createdResult?.raw_key ? (
              <div className="rounded-md border bg-muted/40 p-3">
                <div className="text-xs text-muted-foreground mb-1">新 API 密钥</div>
                <div className="flex items-center gap-2">
                  <code className="flex-1 text-xs break-all">{createdResult.raw_key}</code>
                  <Button size="icon" variant="outline" onClick={() => void handleCopy(createdResult!.raw_key)}>
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ) : null}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>关闭</Button>
            {!createdResult ? (
              <Button onClick={() => void handleCreate()} disabled={createLoading}>
                {createLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                创建
              </Button>
            ) : null}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
