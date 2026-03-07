import { useEffect, useMemo, useState } from "react";
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
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import { parseJSONObjectText } from "@/lib/format";
import type { Account, AccountType, JsonObject } from "@/types";

interface AccountFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialData?: Account | null;
  onSuccess: () => void;
  defaultTypeKey?: string;
}

type SchemaProperty = {
  type?: string;
  title?: string;
  description?: string;
  enum?: Array<string | number | boolean>;
  default?: unknown;
};

export function AccountForm({
  open,
  onOpenChange,
  initialData,
  onSuccess,
  defaultTypeKey,
}: AccountFormProps) {
  const [loading, setLoading] = useState(false);
  const [types, setTypes] = useState<AccountType[]>([]);
  const [form, setForm] = useState({
    typeKey: "",
    identifier: "",
    status: "0",
    spec: "{}",
    tags: "",
  });
  const [specForm, setSpecForm] = useState<Record<string, unknown>>({});

  useEffect(() => {
    if (!open) return;
    void loadTypes();
    if (initialData) {
      setForm({
        typeKey: initialData.type_key,
        identifier: initialData.identifier,
        status: String(initialData.status),
        spec: JSON.stringify(initialData.spec, null, 2),
        tags: initialData.tags?.join(", ") || "",
      });
      setSpecForm((initialData.spec as Record<string, unknown>) ?? {});
    } else {
      setForm({
        typeKey: defaultTypeKey ?? "",
        identifier: "",
        status: "0",
        spec: "{}",
        tags: "",
      });
      setSpecForm({});
    }
  }, [open, initialData, defaultTypeKey]);

  const loadTypes = async () => {
    try {
      setTypes((await api.listAccountTypes()).filter((item) => item.category === "generic"));
    } catch (error) {
      toast.error("加载账号类型失败：" + extractErrorMessage(error));
    }
  };

  const selectedType = useMemo(
    () => types.find((type) => type.key === form.typeKey),
    [types, form.typeKey]
  );

  const schemaProperties = useMemo(() => {
    const schema = selectedType?.schema as JsonObject | undefined;
    const props = schema && typeof schema === "object" ? (schema.properties as Record<string, SchemaProperty> | undefined) : undefined;
    return props && typeof props === "object" ? props : null;
  }, [selectedType]);

  const schemaRequired = useMemo(() => {
    const schema = selectedType?.schema as JsonObject | undefined;
    const required = schema && typeof schema === "object" ? (schema.required as string[] | undefined) : undefined;
    return Array.isArray(required) ? required : [];
  }, [selectedType]);

  useEffect(() => {
    if (!open || initialData) return;
    if (!schemaProperties) return;
    const defaults: Record<string, unknown> = {};
    for (const [key, definition] of Object.entries(schemaProperties)) {
      if (definition.default !== undefined) {
        defaults[key] = definition.default;
      }
    }
    setSpecForm((prev) => ({ ...defaults, ...prev }));
  }, [schemaProperties, open, initialData]);

  const updateSpecValue = (key: string, value: unknown) => {
    setSpecForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      const payload = {
        status: Number(form.status),
        tags: form.tags.split(",").map((t) => t.trim()).filter(Boolean),
        spec: schemaProperties ? (specForm as Record<string, unknown>) : parseJSONObjectText(form.spec, "spec"),
      };

      if (initialData) {
        await api.patchAccount(initialData.id, payload);
        toast.success("账号已更新");
      } else {
        await api.createAccount({
          type_key: form.typeKey,
          identifier: form.identifier.trim(),
          ...payload,
        });
        toast.success("账号已创建");
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
          <SheetTitle>{initialData ? "编辑账号" : "创建账号"}</SheetTitle>
          <SheetDescription>
            {initialData
              ? `正在编辑 ${initialData.id}`
              : "创建一个新的通用账号资产。"}
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            <form id="account-form" onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="acc-status">状态</Label>
                  <Select
                    value={form.status}
                    onValueChange={(v) => setForm({ ...form, status: v })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="0">待验证 (0)</SelectItem>
                      <SelectItem value="1">已验证 (1)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="acc-type">账号类型</Label>
                <Select
                  value={form.typeKey}
                  onValueChange={(v) => setForm({ ...form, typeKey: v })}
                  disabled={!!initialData || !!defaultTypeKey}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {types.map((t) => (
                      <SelectItem key={t.key} value={t.key}>
                        {t.name} ({t.key})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">此处仅支持 generic 类型账号。</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="acc-identifier">标识符</Label>
                <Input
                  id="acc-identifier"
                  value={form.identifier}
                  onChange={(e) => setForm({ ...form, identifier: e.target.value })}
                  placeholder="唯一ID / 用户名"
                  disabled={!!initialData}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="acc-tags">标签（逗号分隔）</Label>
                <Input
                  id="acc-tags"
                  value={form.tags}
                  onChange={(e) => setForm({ ...form, tags: e.target.value })}
                  placeholder="标签1, 标签2"
                />
              </div>

              {schemaProperties ? (
                <div className="space-y-4">
                  <div className="text-sm font-medium">Spec（Schema）</div>
                  {Object.entries(schemaProperties).map(([key, definition]) => {
                    const fieldType = definition.type ?? "string";
                    const label = definition.title ?? key;
                    const description = definition.description;
                    const required = schemaRequired.includes(key);
                    const value = specForm[key];

                    if (definition.enum && definition.enum.length > 0) {
                      return (
                        <div key={key} className="space-y-2">
                          <Label>{label}</Label>
                          <Select
                            value={value ? String(value) : ""}
                            onValueChange={(v) => updateSpecValue(key, v)}
                          >
                            <SelectTrigger>
                              <SelectValue placeholder="请选择" />
                            </SelectTrigger>
                            <SelectContent>
                              {definition.enum.map((option) => (
                                <SelectItem key={String(option)} value={String(option)}>
                                  {String(option)}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          {description ? (
                            <p className="text-xs text-muted-foreground">{description}</p>
                          ) : null}
                        </div>
                      );
                    }

                    if (fieldType === "boolean") {
                      return (
                        <div key={key} className="flex items-center justify-between rounded-md border border-border/60 px-3 py-2">
                          <div>
                            <p className="text-sm font-medium">{label}</p>
                            {description ? (
                              <p className="text-xs text-muted-foreground">{description}</p>
                            ) : null}
                          </div>
                          <Switch
                            checked={Boolean(value)}
                            onCheckedChange={(checked) => updateSpecValue(key, checked)}
                          />
                        </div>
                      );
                    }

                    if (fieldType === "number" || fieldType === "integer") {
                      return (
                        <div key={key} className="space-y-2">
                          <Label>{label}</Label>
                          <Input
                            type="number"
                            value={value === undefined ? "" : String(value)}
                            onChange={(e) => {
                              const next = e.target.value;
                              updateSpecValue(key, next === "" ? undefined : Number(next));
                            }}
                            required={required}
                          />
                          {description ? (
                            <p className="text-xs text-muted-foreground">{description}</p>
                          ) : null}
                        </div>
                      );
                    }

                    if (fieldType === "object" || fieldType === "array") {
                      return (
                        <div key={key} className="space-y-2">
                          <Label>{label}</Label>
                          <Textarea
                            value={value ? JSON.stringify(value, null, 2) : ""}
                            onChange={(e) => {
                              try {
                                updateSpecValue(key, JSON.parse(e.target.value));
                              } catch {
                                updateSpecValue(key, e.target.value);
                              }
                            }}
                            placeholder="JSON"
                            className="font-mono text-xs min-h-[120px]"
                          />
                          {description ? (
                            <p className="text-xs text-muted-foreground">{description}</p>
                          ) : null}
                        </div>
                      );
                    }

                    return (
                      <div key={key} className="space-y-2">
                        <Label>{label}</Label>
                        <Input
                          value={value === undefined ? "" : String(value)}
                          onChange={(e) => updateSpecValue(key, e.target.value)}
                          required={required}
                        />
                        {description ? (
                          <p className="text-xs text-muted-foreground">{description}</p>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="acc-spec">Spec（JSON 对象）</Label>
                  <Textarea
                    id="acc-spec"
                    className="font-mono text-xs min-h-[200px]"
                    value={form.spec}
                    onChange={(e) => setForm({ ...form, spec: e.target.value })}
                  />
                </div>
              )}
            </form>
          </ScrollArea>
        </div>
        <SheetFooter className="p-6 border-t">
          <Button type="submit" form="account-form" disabled={loading}>
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {initialData ? "保存修改" : "创建账号"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
