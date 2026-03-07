import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { parseJSONOrNull, parseJSONObjectText } from "@/lib/format";
import type { AccountType } from "@/types";

interface AccountTypeFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialData?: AccountType | null;
  onSuccess: () => void;
}

const defaultCapabilities = JSON.stringify(
  {
    actions: [{ key: "REGISTER" }, { key: "VERIFY" }],
  },
  null,
  2
);

export function AccountTypeForm({
  open,
  onOpenChange,
  initialData,
  onSuccess,
}: AccountTypeFormProps) {
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    key: "",
    name: "",
    category: "generic",
    schema: "{}",
    capabilities: defaultCapabilities,
    scriptConfig: "",
  });

  useEffect(() => {
    if (open) {
      if (initialData) {
        setForm({
          key: initialData.key,
          name: initialData.name,
          category: initialData.category,
          schema: JSON.stringify(initialData.schema, null, 2),
          capabilities: JSON.stringify(initialData.capabilities, null, 2),
          scriptConfig: initialData.script_config
            ? JSON.stringify(initialData.script_config, null, 2)
            : "",
        });
      } else {
        setForm({
          key: "",
          name: "",
          category: "generic",
          schema: "{}",
          capabilities: defaultCapabilities,
          scriptConfig: "",
        });
      }
    }
  }, [open, initialData]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      const payload = {
        name: form.name.trim(),
        category: form.category,
        schema: parseJSONObjectText(form.schema, "schema"),
        capabilities: parseJSONObjectText(form.capabilities, "capabilities"),
        script_config: parseJSONOrNull(form.scriptConfig, "script_config"),
      };

      if (initialData) {
        await api.patchAccountType(initialData.key, payload);
        toast.success("账号类型已更新");
      } else {
        await api.createAccountType({
          key: form.key.trim(),
          ...payload,
        });
        toast.success("账号类型已创建");
      }
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
      <SheetContent className="w-[600px] sm:max-w-[600px] flex flex-col p-0">
        <SheetHeader className="p-6 border-b">
          <SheetTitle>{initialData ? "编辑类型" : "创建类型"}</SheetTitle>
          <SheetDescription>
            {initialData
              ? `编辑 ${initialData.key} 的配置。`
              : "填写基础字段后即可在后端自动生成对应 octoModule 脚本入口。"}
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            <form id="account-type-form" onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="type-key">键名</Label>
                <Input
                  id="type-key"
                  value={form.key}
                  onChange={(e) => setForm({ ...form, key: e.target.value })}
                  placeholder="generic_x"
                  disabled={!!initialData}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="type-name">名称</Label>
                <Input
                  id="type-name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="Generic X"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="type-category">分类</Label>
                <Select
                  value={form.category}
                  onValueChange={(v) => setForm({ ...form, category: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="generic">generic</SelectItem>
                    <SelectItem value="email">email</SelectItem>
                    <SelectItem value="system">system</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  仅 generic 类型会自动创建 Octo 模块脚本入口。
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="type-schema">Schema（JSON 对象）</Label>
                <Textarea
                  id="type-schema"
                  className="font-mono text-xs min-h-[100px]"
                  value={form.schema}
                  onChange={(e) => setForm({ ...form, schema: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="type-capabilities">能力配置（JSON 对象）</Label>
                <Textarea
                  id="type-capabilities"
                  className="font-mono text-xs min-h-[100px]"
                  value={form.capabilities}
                  onChange={(e) => setForm({ ...form, capabilities: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="type-script-config">脚本配置（JSON / null）</Label>
                <Textarea
                  id="type-script-config"
                  className="font-mono text-xs min-h-[100px]"
                  value={form.scriptConfig}
                  onChange={(e) => setForm({ ...form, scriptConfig: e.target.value })}
                  placeholder='{"octoModule":{"entry":"social/discord.py"}}'
                />
              </div>
            </form>
          </ScrollArea>
        </div>
        <SheetFooter className="p-6 border-t">
          <Button type="submit" form="account-type-form" disabled={loading}>
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {initialData ? "保存修改" : "立即创建"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
