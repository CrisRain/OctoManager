export interface FieldOption {
  label: string;
  value: string;
  description?: string;
}

export interface FieldConfig {
  name: string;
  label: string;
  type: "text" | "textarea" | "select" | "password" | "number" | "switch" | "tags";
  placeholder?: string;
  description?: string;
  required?: boolean;
  defaultValue?: any;
  options?: FieldOption[];
  min?: number;
  max?: number;
  rows?: number;
  autoSuggest?: {
    type: "timestamp" | "uuid" | "random" | "currentUser";
    format?: string;
  };
  /** 悬浮提示文本，显示在字段标签旁的 ? 图标上 */
  help?: string;
}
