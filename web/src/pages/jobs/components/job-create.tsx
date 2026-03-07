import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Pagination } from "@/components/ui/pagination";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, parseJSONObjectText } from "@/lib/format";
import type { Account, AccountType, JsonObject } from "@/types";

interface JobCreateProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

const ACCOUNT_PAGE_SIZE = 10;

function extractActionKeys(capabilities: JsonObject): string[] {
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

export function JobCreate({ open, onOpenChange, onSuccess }: JobCreateProps) {
  const [loading, setLoading] = useState(false);
  const [types, setTypes] = useState<AccountType[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [accountsTotal, setAccountsTotal] = useState(0);
  const [accountsOffset, setAccountsOffset] = useState(0);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [selectedAccountIDs, setSelectedAccountIDs] = useState<number[]>([]);
  const [form, setForm] = useState({
    typeKey: "",
    actionKey: "VERIFY",
    selector: "{}",
    params: "{}"
  });

  const selectedType = useMemo(
    () => types.find((item) => item.key === form.typeKey) ?? null,
    [types, form.typeKey]
  );
  const availableActions = useMemo(
    () => (selectedType ? extractActionKeys(selectedType.capabilities) : []),
    [selectedType]
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    setAccountsOffset(0);
    setSelectedAccountIDs([]);
    setForm({
      typeKey: "",
      actionKey: "VERIFY",
      selector: "{}",
      params: "{}"
    });
    void loadTypes();
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    void loadAccounts();
  }, [open, accountsOffset, form.typeKey]);

  useEffect(() => {
    if (!open) {
      return;
    }
    setAccountsOffset(0);
    setSelectedAccountIDs([]);
  }, [open, form.typeKey]);

  useEffect(() => {
    if (!availableActions.length) {
      return;
    }
    if (!availableActions.includes(form.actionKey)) {
      setForm((prev) => ({
        ...prev,
        actionKey: availableActions[0]
      }));
    }
  }, [availableActions, form.actionKey]);

  const loadTypes = async () => {
    try {
      const res = (await api.listAccountTypes()).filter((item) => item.category === "generic");
      setTypes(res);
      if (res.length > 0) {
        const firstType = res[0];
        const actionCandidates = extractActionKeys(firstType.capabilities);
        setForm((prev) => ({
          ...prev,
          typeKey: prev.typeKey || firstType.key,
          actionKey: prev.actionKey || actionCandidates[0] || "VERIFY"
        }));
      }
    } catch (error) {
      toast.error("加载账号类型失败: " + extractErrorMessage(error));
    }
  };

  const loadAccounts = async () => {
    try {
      setAccountsLoading(true);
      const res = await api.listAccounts({
        limit: ACCOUNT_PAGE_SIZE,
        offset: accountsOffset,
        type_key: form.typeKey || undefined
      });
      setAccounts(res.items);
      setAccountsTotal(res.total);
    } catch (error) {
      toast.error("加载账号列表失败: " + extractErrorMessage(error));
    } finally {
      setAccountsLoading(false);
    }
  };

  const toggleAccount = (account: Account, checked: boolean) => {
    if (form.typeKey && account.type_key !== form.typeKey) {
      return;
    }

    setSelectedAccountIDs((prev) => {
      if (checked) {
        if (prev.includes(account.id)) {
          return prev;
        }
        return [...prev, account.id];
      }
      return prev.filter((id) => id !== account.id);
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      const selector = parseJSONObjectText(form.selector, "selector");
      if (selectedAccountIDs.length > 0) {
        selector.account_ids = selectedAccountIDs;
      }

      await api.createJob({
        type_key: form.typeKey.trim(),
        action_key: form.actionKey.trim(),
        selector,
        params: parseJSONObjectText(form.params, "params")
      });
      toast.success("任务已创建");
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[500px] sm:max-w-[500px]">
        <SheetHeader>
          <SheetTitle>创建任务</SheetTitle>
          <SheetDescription>
            创建一个新的后台任务。这里只允许 generic account type，对应 Octo 模块分发执行。
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="typeKey">类型键</Label>
            <Select
              value={form.typeKey}
              onValueChange={(value) => setForm({ ...form, typeKey: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择类型" />
              </SelectTrigger>
              <SelectContent>
                {types.map((type) => (
                  <SelectItem key={type.key} value={type.key}>
                    {type.key}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">此处仅支持 generic 类型账号。</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="actionKey">动作键</Label>
            {availableActions.length > 0 ? (
              <Select
                value={form.actionKey}
                onValueChange={(value) => setForm((prev) => ({ ...prev, actionKey: value }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择动作" />
                </SelectTrigger>
                <SelectContent>
                  {availableActions.map((action) => (
                    <SelectItem key={action} value={action}>
                      {action}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Input
                id="actionKey"
                value={form.actionKey}
                onChange={(e) => setForm({ ...form, actionKey: e.target.value })}
                required
              />
            )}
            <p className="text-xs text-muted-foreground">
              {availableActions.length > 0
                ? "已根据 Account Type 的 capabilities.actions 加载可选动作，避免手动输入错误。"
                : "当前类型未声明 actions，可手动输入 Action Key。"}
            </p>
          </div>

          <div className="space-y-2">
            <Label>目标账号（可多选，可选填）</Label>
            <div className="rounded-md border p-3 space-y-3">
              {accountsLoading ? (
                <div className="text-sm text-muted-foreground">加载账号中...</div>
              ) : accounts.length === 0 ? (
                <div className="text-sm text-muted-foreground">暂无可选账号</div>
              ) : (
                <div className="max-h-44 overflow-y-auto space-y-2 pr-1">
                  {accounts.map((account) => {
                    const checked = selectedAccountIDs.includes(account.id);
                    const disabled = Boolean(form.typeKey) && account.type_key !== form.typeKey;
                    return (
                      <label
                        key={account.id}
                        className={`flex items-start gap-2 rounded border px-2 py-1.5 ${
                          disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"
                        }`}
                      >
                        <Checkbox
                          checked={checked}
                          disabled={disabled}
                          onCheckedChange={(next) => toggleAccount(account, Boolean(next))}
                        />
                        <span className="min-w-0 flex-1 text-xs">
                          <span className="block truncate text-foreground">
                            {account.identifier}
                          </span>
                          <span className="block text-muted-foreground font-mono">
                            {compactId(account.id)} · {account.type_key}
                          </span>
                        </span>
                      </label>
                    );
                  })}
                </div>
              )}
              <Pagination
                total={accountsTotal}
                limit={ACCOUNT_PAGE_SIZE}
                offset={accountsOffset}
                onPageChange={setAccountsOffset}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              已选择 {selectedAccountIDs.length} 个账号，提交时会自动写入 <code>selector.account_ids</code>。
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="selector">选择器（高级 JSON）</Label>
            <Textarea
              id="selector"
              className="font-mono text-xs"
              rows={4}
              value={form.selector}
              onChange={(e) => setForm({ ...form, selector: e.target.value })}
            />
            <p className="text-xs text-muted-foreground">
              可配置 <code>identifier_contains</code> / <code>limit</code> 等进阶条件。若已选择 Target Accounts，将优先覆盖 <code>account_ids</code>。
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="params">参数（JSON）</Label>
            <Textarea
              id="params"
              className="font-mono text-xs"
              rows={4}
              value={form.params}
              onChange={(e) => setForm({ ...form, params: e.target.value })}
            />
          </div>

          <SheetFooter>
            <Button type="submit" disabled={loading}>
              {loading ? "创建中..." : "确认创建"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
