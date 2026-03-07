import {
  Activity,
  Database,
  KeyRound,
  Layers,
  Link2,
  Mail,
  Menu,
  Settings,
  Shapes,
  Workflow,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { AccountType } from "@/types";

type NavItem = {
  to: string;
  label: string;
  icon: typeof Activity;
  children?: Array<{ to: string; label: string }>;
};

export function AppShell() {
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const [accountTypes, setAccountTypes] = useState<AccountType[]>([]);

  useEffect(() => {
    let mounted = true;
    api
      .listAccountTypes()
      .then((items) => {
        if (mounted) {
          setAccountTypes(items);
        }
      })
      .catch(() => {
        if (mounted) {
          setAccountTypes([]);
        }
      });
    return () => {
      mounted = false;
    };
  }, []);

  const navItems = useMemo<NavItem[]>(() => {
    const genericTypes = accountTypes.filter((item) => item.category === "generic");
    const accountChildren = [
      { to: "/accounts", label: "全部账号" },
      ...genericTypes.map((item) => ({
        to: `/accounts/${item.key}`,
        label: item.name,
      })),
    ];
    const emailChildren = [{ to: "/email-accounts/outlook", label: "Outlook 邮箱管理" }];

    return [
      { to: "/dashboard", label: "控制台", icon: Activity },
      { to: "/account-types", label: "账号类型", icon: Shapes },
      { to: "/accounts", label: "账号管理", icon: Database, children: accountChildren },
      { to: "/email-accounts", label: "邮箱账号", icon: Mail, children: emailChildren },
      { to: "/jobs", label: "任务", icon: Workflow },
      { to: "/modules", label: "Octo 模块", icon: Layers },
      { to: "/api-keys", label: "API 密钥", icon: KeyRound },
      { to: "/triggers", label: "触发器", icon: Link2 },
      { to: "/settings", label: "设置", icon: Settings },
    ];
  }, [accountTypes]);

  const currentTitle = useMemo(() => {
    const path = location.pathname;
    const topMatch = navItems.find((item) => path === item.to || path.startsWith(`${item.to}/`));
    if (!topMatch) {
      return "控制台";
    }
    const childMatch = topMatch.children?.find((child) => path === child.to);
    return childMatch?.label ?? topMatch.label;
  }, [location.pathname, navItems]);

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-50 flex h-14 items-center gap-4 border-b bg-card px-4 lg:hidden">
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="icon" className="shrink-0">
              <Menu className="h-5 w-5" />
              <span className="sr-only">切换导航菜单</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="flex w-[280px] flex-col p-0 sm:w-[300px]">
            <div className="flex h-14 items-center border-b px-6 font-semibold">
              <span className="text-lg font-bold">OctoManager</span>
            </div>
            <ScrollArea className="flex-1 py-4">
              <nav className="grid gap-1 px-4 text-sm font-medium">
                {navItems.map((item) => {
                  const Icon = item.icon;
                  const hasChildren = item.children && item.children.length > 0;
                  const isAncestor = hasChildren && location.pathname.startsWith(`${item.to}/`);
                  return (
                    <div key={item.to} className="grid gap-1">
                      <NavLink
                        to={item.to}
                        onClick={() => setOpen(false)}
                        end
                        className={({ isActive }) =>
                          cn(
                            "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-all",
                            isActive
                              ? "bg-foreground font-medium text-background"
                              : isAncestor
                                ? "font-medium text-foreground hover:bg-muted"
                                : "text-muted-foreground hover:bg-muted hover:text-foreground",
                          )
                        }
                      >
                        <Icon className="h-4 w-4" />
                        {item.label}
                      </NavLink>
                      {hasChildren ? (
                        <div className="ml-7 grid gap-1">
                          {item.children!.map((child) => (
                            <NavLink
                              key={child.to}
                              to={child.to}
                              onClick={() => setOpen(false)}
                              end
                              className={({ isActive }) =>
                                cn(
                                  "flex items-center gap-2 rounded-md px-3 py-1.5 text-xs transition-all",
                                  isActive
                                    ? "bg-muted font-medium text-foreground"
                                    : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
                                )
                              }
                            >
                              <span className="h-1.5 w-1.5 rounded-full bg-current opacity-60" />
                              {child.label}
                            </NavLink>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  );
                })}
              </nav>
            </ScrollArea>
          </SheetContent>
        </Sheet>
        <div className="flex flex-1 items-center justify-between">
          <span className="font-semibold">OctoManager</span>
        </div>
      </header>

      <div className="flex flex-1">
        <aside className="sticky top-0 hidden h-screen w-[240px] flex-col border-r bg-card lg:flex">
          <div className="flex h-14 items-center border-b px-6">
            <span className="text-lg font-bold tracking-tight">OctoManager</span>
          </div>
          <ScrollArea className="flex-1 py-4">
            <nav className="grid gap-1 px-4 text-sm font-medium">
              {navItems.map((item) => {
                const Icon = item.icon;
                const hasChildren = item.children && item.children.length > 0;
                const isAncestor = hasChildren && location.pathname.startsWith(`${item.to}/`);
                return (
                  <div key={item.to} className="grid gap-1">
                    <NavLink
                      to={item.to}
                      end
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-all",
                          isActive
                            ? "bg-foreground font-medium text-background"
                            : isAncestor
                              ? "font-medium text-foreground hover:bg-muted"
                              : "text-muted-foreground hover:bg-muted hover:text-foreground",
                        )
                      }
                    >
                      <Icon className="h-4 w-4" />
                      {item.label}
                    </NavLink>
                    {hasChildren ? (
                      <div className="ml-7 grid gap-1">
                        {item.children!.map((child) => (
                          <NavLink
                            key={child.to}
                            to={child.to}
                            end
                            className={({ isActive }) =>
                              cn(
                                "flex items-center gap-2 rounded-md px-3 py-1.5 text-xs transition-all",
                                isActive
                                  ? "bg-muted font-medium text-foreground"
                                  : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
                              )
                            }
                          >
                            <span className="h-1.5 w-1.5 rounded-full bg-current opacity-60" />
                            {child.label}
                          </NavLink>
                        ))}
                      </div>
                    ) : null}
                  </div>
                );
              })}
            </nav>
          </ScrollArea>
          <div className="border-t p-4">
            <div className="rounded-md border bg-muted/50 p-3">
              <div className="flex items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-full bg-foreground">
                  <Activity className="h-3.5 w-3.5 text-background" />
                </div>
                <div>
                  <p className="text-xs font-medium">状态：在线</p>
                  <p className="text-[10px] text-muted-foreground">v1.0.0</p>
                </div>
              </div>
            </div>
          </div>
        </aside>

        <main className="flex-1">
          <div className="container mx-auto max-w-7xl space-y-4 p-4 lg:p-8">
            <div className="mb-4 flex items-center justify-between lg:hidden">
              <h1 className="text-lg font-semibold md:text-2xl">{currentTitle}</h1>
            </div>
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
