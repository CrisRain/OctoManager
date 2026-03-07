import { useCallback, useEffect, useMemo, useState } from "react";
import { Edit, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useParams } from "react-router-dom";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { Account } from "@/types";
import { AccountForm } from "./accounts/components/account-form";

export function AccountsPage() {
  const { typeKey } = useParams();
  const [items, setItems] = useState<Account[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<Account | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [batchLoading, setBatchLoading] = useState(false);
  const [typeLabel, setTypeLabel] = useState<string | null>(null);

  useEffect(() => {
    setOffset(0);
  }, [typeKey]);

  useEffect(() => {
    if (!typeKey) {
      setTypeLabel(null);
      return;
    }
    let mounted = true;
    api
      .getAccountType(typeKey)
      .then((res) => {
        if (mounted) {
          setTypeLabel(res.name);
        }
      })
      .catch(() => {
        if (mounted) {
          setTypeLabel(null);
        }
      });
    return () => {
      mounted = false;
    };
  }, [typeKey]);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listAccounts({ limit: pageSize, offset, type_key: typeKey });
      setItems(res.items);
      setTotal(res.total);
      setSelectedIds(new Set());
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [offset, pageSize, typeKey]);

  const handleLimitChange = (newLimit: number) => {
    setOffset(0);
    setPageSize(newLimit);
  };

  useEffect(() => {
    void load();
  }, [load]);

  const handleDelete = async (id: number) => {
    if (!confirm("确定要删除此账号？")) return;
    try {
      await api.deleteAccount(id);
      toast.success("账号已删除");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleCreate = () => {
    setEditingItem(null);
    setFormOpen(true);
  };

  const handleEdit = (item: Account) => {
    setEditingItem(item);
    setFormOpen(true);
  };

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === items.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(items.map((i) => i.id)));
    }
  };

  const handleBatchDelete = async () => {
    if (!confirm(`确定要删除选中的 ${selectedIds.size} 个账号？`)) return;
    setBatchLoading(true);
    try {
      const result = await api.batchDeleteAccounts([...selectedIds]);
      if (result.queued) {
        toast.success(
          `批量删除已入队${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`
        );
      } else if (result.failed > 0) {
        toast.warning(`已删除 ${result.success} 个，失败 ${result.failed} 个`);
      } else {
        toast.success(`已删除 ${result.success} 个账号`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  const handleBatchSetStatus = async (status: number) => {
    setBatchLoading(true);
    try {
      const result = await api.batchPatchAccounts({ ids: [...selectedIds], status });
      if (result.queued) {
        toast.success(
          `批量更新已入队${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`
        );
      } else if (result.failed > 0) {
        toast.warning(`已更新 ${result.success} 个，失败 ${result.failed} 个`);
      } else {
        toast.success(`已更新 ${result.success} 个账号`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  const allSelected = items.length > 0 && selectedIds.size === items.length;
  const someSelected = selectedIds.size > 0;

  const title = useMemo(() => {
    if (!typeKey) return "账号";
    return typeLabel ? `${typeLabel} 账号` : `账号 / ${typeKey}`;
  }, [typeKey, typeLabel]);

  const description = useMemo(() => {
    if (!typeKey) return "管理所有账号资源。";
    return `管理 ${typeLabel ?? typeKey} 的账号。`;
  }, [typeKey, typeLabel]);

  return (
    <div className="space-y-4">
      <PageHeader
        title={title}
        description={description}
        action={
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            创建账号
          </Button>
        }
      />

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle>账号列表</CardTitle>
            <CardDescription>共 {total} 条</CardDescription>
          </div>
          <Button variant="outline" size="icon" onClick={() => void load()} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </CardHeader>

        <div className="mx-6 mb-3 flex items-center gap-2 rounded-lg border border-border bg-muted/40 px-4 py-2 text-sm">
          <span className="text-muted-foreground">
            {someSelected ? `已选 ${selectedIds.size} 项` : "批量操作"}
          </span>
          <div className="ml-auto flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={batchLoading || !someSelected}
              onClick={() => void handleBatchSetStatus(1)}
            >
              启用
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={batchLoading || !someSelected}
              onClick={() => void handleBatchSetStatus(0)}
            >
              禁用
            </Button>
            <Button
              size="sm"
              variant="destructive"
              disabled={batchLoading || !someSelected}
              onClick={() => void handleBatchDelete()}
            >
              <Trash2 className="mr-1 h-3.5 w-3.5" />
              删除
            </Button>
          </div>
        </div>

        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground">加载中...</div>
          ) : items.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无账号
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10">
                    <Checkbox
                      checked={allSelected}
                      onCheckedChange={toggleSelectAll}
                      aria-label="全选"
                    />
                  </TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>标识符</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>标签</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow
                    key={item.id}
                    data-state={selectedIds.has(item.id) ? "selected" : undefined}
                  >
                    <TableCell>
                      <Checkbox
                        checked={selectedIds.has(item.id)}
                        onCheckedChange={() => toggleSelect(item.id)}
                        aria-label={`选择 ${item.identifier}`}
                      />
                    </TableCell>
                    <TableCell className="font-medium font-mono text-xs">{compactId(item.id)}</TableCell>
                    <TableCell>{item.type_key}</TableCell>
                    <TableCell>{item.identifier}</TableCell>
                    <TableCell>
                      <Badge variant={item.status === 1 ? "outline" : "secondary"}>
                        {item.status === 1 ? "已启用" : "已禁用"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {item.tags?.map((tag) => (
                          <Badge key={tag} variant="secondary" className="h-5 px-1 py-0 text-[10px]">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDateTime(item.updated_at)}
                    </TableCell>
                    <TableCell className="space-x-2 text-right">
                      <Button variant="ghost" size="icon" onClick={() => handleEdit(item)} title="编辑">
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => void handleDelete(item.id)}
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
          <Pagination
            total={total}
            limit={pageSize}
            offset={offset}
            onPageChange={setOffset}
            onLimitChange={handleLimitChange}
          />
        </CardContent>
      </Card>

      <AccountForm
        open={formOpen}
        onOpenChange={setFormOpen}
        initialData={editingItem}
        onSuccess={() => void load()}
        defaultTypeKey={typeKey}
      />
    </div>
  );
}
