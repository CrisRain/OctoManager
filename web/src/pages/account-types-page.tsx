import { useCallback, useEffect, useState } from "react";
import { Edit, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { JsonView } from "@/components/json-view";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { AccountType } from "@/types";
import { AccountTypeForm } from "./account-types/components/account-type-form";

export function AccountTypesPage() {
  const [items, setItems] = useState<AccountType[]>([]);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<AccountType | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setItems(await api.listAccountTypes());
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleDelete = async (key: string) => {
    if (!confirm(`确定要删除账号类型 ${key} 吗？`)) return;
    try {
      await api.deleteAccountType(key);
      toast.success(`已删除 ${key}`);
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleCreate = () => {
    setEditingItem(null);
    setFormOpen(true);
  };

  const handleEdit = (item: AccountType) => {
    setEditingItem(item);
    setFormOpen(true);
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="账号类型"
        description="定义账号类型能力、Schema 与脚本配置。"
        action={
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            创建类型
          </Button>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>类型列表</CardTitle>
          <CardDescription>总数 {items.length}</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground">加载中...</div>
          ) : items.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无类型
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>键名</TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>分类</TableHead>
                  <TableHead>版本</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium font-mono text-xs">{item.key}</TableCell>
                    <TableCell>{item.name}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">{item.category}</Badge>
                    </TableCell>
                    <TableCell>{item.version}</TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {formatDateTime(item.updated_at)}
                    </TableCell>
                    <TableCell className="text-right space-x-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleEdit(item)}
                        title="编辑"
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => handleDelete(item.key)}
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

          {items.length > 0 && (
            <div className="mt-6 grid gap-6 md:grid-cols-2">
              <div>
                <p className="mb-2 text-sm font-medium text-muted-foreground">Schema 示例 (第一个)</p>
                <div className="rounded-md border bg-muted/50 p-2">
                  <JsonView value={items[0].schema} />
                </div>
              </div>
              <div>
                <p className="mb-2 text-sm font-medium text-muted-foreground">Capabilities 示例 (第一个)</p>
                <div className="rounded-md border bg-muted/50 p-2">
                  <JsonView value={items[0].capabilities} />
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <AccountTypeForm
        open={formOpen}
        onOpenChange={setFormOpen}
        initialData={editingItem}
        onSuccess={() => void load()}
      />
    </div>
  );
}
