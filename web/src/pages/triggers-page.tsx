import { useCallback, useEffect, useMemo, useState } from "react";
import { Copy, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import type { AccountType, CreateTriggerResult, TriggerEndpoint, TriggerMode } from "@/types";

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/+$/, "") ?? "";

type TriggerFormState = {
  name: string;
  slug: string;
  typeKey: string;
  actionKey: string;
  mode: TriggerMode;
  defaultSelector: string;
  defaultParams: string;
  enabled: boolean;
};

const DEFAULT_FORM: TriggerFormState = {
  name: "",
  slug: "",
  typeKey: "",
  actionKey: "",
  mode: "async",
  defaultSelector: "{}",
  defaultParams: "{}",
  enabled: true,
};

function extractActionKeys(capabilities: AccountType["capabilities"]): string[] {
  const rawActions = capabilities.actions;
  if (!Array.isArray(rawActions)) {
    return [];
  }

  const keys = new Set<string>();
  for (const raw of rawActions) {
    if (typeof raw === "string") {
      const key = raw.trim();
      if (key) {
        keys.add(key);
      }
      continue;
    }

    if (!raw || typeof raw !== "object") {
      continue;
    }

    const key = String((raw as Record<string, unknown>).key ?? "").trim();
    if (key) {
      keys.add(key);
    }
  }

  return Array.from(keys);
}

function buildDefaultForm(genericTypes: AccountType[]): TriggerFormState {
  const firstType = genericTypes[0];
  const firstAction = firstType ? extractActionKeys(firstType.capabilities)[0] ?? "" : "";

  return {
    ...DEFAULT_FORM,
    typeKey: firstType?.key ?? "",
    actionKey: firstAction,
  };
}

function parseJSONObjectText(value: string, fieldName: string): Record<string, unknown> {
  const raw = value.trim() || "{}";

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error(`${fieldName} 不是合法 JSON`);
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`${fieldName} 必须是 JSON 对象`);
  }

  return parsed as Record<string, unknown>;
}

export function TriggersPage() {
  const [items, setItems] = useState<TriggerEndpoint[]>([]);
  const [types, setTypes] = useState<AccountType[]>([]);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<TriggerEndpoint | null>(null);
  const [form, setForm] = useState<TriggerFormState>(DEFAULT_FORM);
  const [saving, setSaving] = useState(false);
  const [createdResult, setCreatedResult] = useState<CreateTriggerResult | null>(null);

  const genericTypes = useMemo(
    () => types.filter((item) => item.category === "generic"),
    [types]
  );
  const selectedType = useMemo(
    () => genericTypes.find((item) => item.key === form.typeKey) ?? null,
    [genericTypes, form.typeKey]
  );
  const suggestedActions = useMemo(
    () => (selectedType ? extractActionKeys(selectedType.capabilities) : []),
    [selectedType]
  );

  const webhookBase = useMemo(() => {
    if (!API_BASE) {
      return "/webhooks";
    }
    return `${API_BASE}/webhooks`;
  }, []);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const [triggerItems, accountTypes] = await Promise.all([
        api.listTriggers(),
        api.listAccountTypes(),
      ]);
      setItems(triggerItems);
      setTypes(accountTypes);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!formOpen || editing || form.typeKey || genericTypes.length === 0) {
      return;
    }

    const defaults = buildDefaultForm(genericTypes);
    setForm((current) => ({
      ...current,
      typeKey: defaults.typeKey,
      actionKey: current.actionKey || defaults.actionKey,
    }));
  }, [editing, form.actionKey, form.typeKey, formOpen, genericTypes]);

  useEffect(() => {
    if (!formOpen || editing || suggestedActions.length === 0 || form.actionKey.trim()) {
      return;
    }

    setForm((current) => ({
      ...current,
      actionKey: suggestedActions[0],
    }));
  }, [editing, form.actionKey, formOpen, suggestedActions]);

  const updateForm = (patch: Partial<TriggerFormState>) => {
    setForm((current) => ({ ...current, ...patch }));
  };

  const handleDialogChange = (open: boolean) => {
    setFormOpen(open);
    if (!open) {
      setCreatedResult(null);
      setEditing(null);
    }
  };

  const openCreate = () => {
    setEditing(null);
    setCreatedResult(null);
    setForm(buildDefaultForm(genericTypes));
    setFormOpen(true);
  };

  const openEdit = (item: TriggerEndpoint) => {
    setEditing(item);
    setCreatedResult(null);
    setForm({
      name: item.name,
      slug: item.slug,
      typeKey: item.type_key,
      actionKey: item.action_key,
      mode: item.mode,
      defaultSelector: JSON.stringify(item.default_selector ?? {}, null, 2),
      defaultParams: JSON.stringify(item.default_params ?? {}, null, 2),
      enabled: item.enabled,
    });
    setFormOpen(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) {
      toast.error("请输入 Trigger 名称");
      return;
    }
    if (!form.slug.trim()) {
      toast.error("请输入 Trigger Slug");
      return;
    }
    if (!form.typeKey.trim()) {
      toast.error(genericTypes.length === 0 ? "请先创建一个 generic 账号类型" : "请选择 generic 账号类型");
      return;
    }
    if (!form.actionKey.trim()) {
      toast.error("请输入 Action Key");
      return;
    }

    let defaultSelector: Record<string, unknown>;
    let defaultParams: Record<string, unknown>;
    try {
      defaultSelector = parseJSONObjectText(form.defaultSelector, "默认 selector");
      defaultParams = parseJSONObjectText(form.defaultParams, "默认 params");
    } catch (error) {
      toast.error(extractErrorMessage(error));
      return;
    }

    setSaving(true);
    try {
      const payload = {
        name: form.name.trim(),
        slug: form.slug.trim(),
        type_key: form.typeKey.trim(),
        action_key: form.actionKey.trim(),
        mode: form.mode,
        default_selector: defaultSelector,
        default_params: defaultParams,
      };

      if (editing) {
        await api.patchTrigger(editing.id, {
          ...payload,
          enabled: form.enabled,
        });
        toast.success("Trigger 已更新");
        setFormOpen(false);
        setEditing(null);
      } else {
        const result = await api.createTrigger(payload);
        setCreatedResult(result);
        toast.success("Trigger 已创建");
      }

      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item: TriggerEndpoint) => {
    if (!window.confirm(`确认删除 Trigger "${item.name}" 吗？`)) {
      return;
    }

    try {
      await api.deleteTrigger(item.id);
      toast.success("Trigger 已删除");
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

  const hasGenericTypes = genericTypes.length > 0;

  return (
    <div className="space-y-4">
      <PageHeader
        title="触发器"
        description="把外部 Webhook 请求接进 generic 模块动作，支持同步和异步两种触发方式。"
      >
        <Button onClick={openCreate}>
          <Plus className="mr-2 h-4 w-4" />
          新建 Trigger
        </Button>
      </PageHeader>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle>Trigger 列表</CardTitle>
            <CardDescription>共 {items.length} 个端点</CardDescription>
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
              暂无 Trigger。
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>Slug</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>动作</TableHead>
                  <TableHead>模式</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>Token 前缀</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="font-mono text-xs">{item.slug}</TableCell>
                    <TableCell>{item.type_key}</TableCell>
                    <TableCell>{item.action_key}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{item.mode}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={item.enabled ? "outline" : "secondary"}>
                        {item.enabled ? "启用" : "禁用"}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{item.token_prefix}</TableCell>
                    <TableCell className="space-x-2 text-right">
                      <Button size="icon" variant="ghost" onClick={() => openEdit(item)} title="编辑">
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
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

      <Dialog open={formOpen} onOpenChange={handleDialogChange}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑 Trigger" : "新建 Trigger"}</DialogTitle>
            <DialogDescription>
              Trigger 只支持 generic 类型账号。同步模式会直接返回模块执行结果，异步模式会进入 Asynq。
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4">
            {!hasGenericTypes && !editing ? (
              <div className="rounded-md border border-dashed border-border/80 bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
                当前没有可用的 generic 账号类型。先到“账号类型”页面创建一个 generic 类型，再回来创建 Trigger。
              </div>
            ) : null}

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>名称</Label>
                <Input
                  value={form.name}
                  onChange={(event) => updateForm({ name: event.target.value })}
                  placeholder="用户注册回调"
                />
              </div>
              <div className="space-y-2">
                <Label>Slug</Label>
                <Input
                  value={form.slug}
                  onChange={(event) => updateForm({ slug: event.target.value })}
                  placeholder="user-register"
                  disabled={!!editing}
                />
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>类型</Label>
                {hasGenericTypes ? (
                  <Select value={form.typeKey} onValueChange={(value) => updateForm({ typeKey: value })}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择 generic 类型" />
                    </SelectTrigger>
                    <SelectContent>
                      {genericTypes.map((item) => (
                        <SelectItem key={item.key} value={item.key}>
                          {item.name} ({item.key})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input value={form.typeKey} placeholder="暂无可用 generic 类型" readOnly disabled />
                )}
                <p className="text-xs text-muted-foreground">这里只允许绑定 generic 账号类型。</p>
              </div>

              <div className="space-y-2">
                <Label>动作</Label>
                <Input
                  value={form.actionKey}
                  onChange={(event) => updateForm({ actionKey: event.target.value })}
                  placeholder="REGISTER"
                />
                {suggestedActions.length > 0 ? (
                  <div className="flex flex-wrap gap-2">
                    {suggestedActions.map((action) => (
                      <Button
                        key={action}
                        type="button"
                        size="sm"
                        variant={form.actionKey.trim() === action ? "secondary" : "outline"}
                        className="h-7 px-2 text-xs"
                        onClick={() => updateForm({ actionKey: action })}
                      >
                        {action}
                      </Button>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    当前类型未声明 capabilities.actions，可手动输入 Action Key。
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <Label>触发方式</Label>
                <Select value={form.mode} onValueChange={(value: TriggerMode) => updateForm({ mode: value })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="async">async</SelectItem>
                    <SelectItem value="sync">sync</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label>默认 selector (JSON)</Label>
              <Textarea
                className="min-h-[96px] font-mono text-xs"
                value={form.defaultSelector}
                onChange={(event) => updateForm({ defaultSelector: event.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label>默认 params (JSON)</Label>
              <Textarea
                className="min-h-[96px] font-mono text-xs"
                value={form.defaultParams}
                onChange={(event) => updateForm({ defaultParams: event.target.value })}
              />
            </div>

            {editing ? (
              <div className="flex items-center gap-3">
                <Switch checked={form.enabled} onCheckedChange={(checked) => updateForm({ enabled: checked })} />
                <span className="text-sm">启用此 Trigger</span>
              </div>
            ) : null}

            {createdResult ? (
              <div className="rounded-md border bg-muted/40 p-3">
                <div className="text-xs text-muted-foreground">Webhook 地址</div>
                <div className="mt-1 flex items-center gap-2">
                  <code className="flex-1 break-all text-xs">{`${webhookBase}/${createdResult.endpoint.slug}`}</code>
                  <Button
                    size="icon"
                    variant="outline"
                    onClick={() => void handleCopy(`${webhookBase}/${createdResult.endpoint.slug}`)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>

                {createdResult.raw_token ? (
                  <>
                    <div className="mt-3 text-xs text-muted-foreground">Trigger Token，仅显示一次</div>
                    <div className="mt-1 flex items-center gap-2">
                      <code className="flex-1 break-all text-xs">{createdResult.raw_token}</code>
                      <Button
                        size="icon"
                        variant="outline"
                        onClick={() => void handleCopy(createdResult.raw_token ?? "")}
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </>
                ) : null}
              </div>
            ) : null}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => handleDialogChange(false)}>
              关闭
            </Button>
            <Button onClick={() => void handleSave()} disabled={saving || (!editing && !hasGenericTypes)}>
              {saving ? "保存中..." : editing ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
