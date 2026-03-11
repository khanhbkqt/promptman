import * as React from "react"
import { Link, Outlet, useLocation } from "react-router-dom"
import {
  FolderOpen,
  Globe,
  FlaskConical,
  Clock,
  Zap,
  LayoutDashboard,
  Settings,
} from "lucide-react"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { ConnectionStatus } from "@/components/ConnectionStatus"
import { EnvSwitcher } from "@/components/env-switcher"

const NAV_ITEMS = [
  {
    title: "Collections",
    url: "/collections",
    icon: FolderOpen,
  },
  {
    title: "Environments",
    url: "/environments",
    icon: Globe,
  },
  {
    title: "Tests",
    url: "/tests",
    icon: FlaskConical,
  },
  {
    title: "History",
    url: "/history",
    icon: Clock,
  },
  {
    title: "Stress",
    url: "/stress",
    icon: Zap,
  },
]

export function AppLayout() {
  const location = useLocation()

  const currentTitle = React.useMemo(() => {
    const item = NAV_ITEMS.find((item) => item.url === location.pathname)
    return item?.title || "PromptMan"
  }, [location.pathname])

  return (
    <SidebarProvider>
      <Sidebar collapsible="icon">
        <SidebarHeader className="h-14 flex items-center justify-start px-4">
          <Link to="/collections" className="flex items-center gap-2 font-bold text-xl">
            <LayoutDashboard className="w-6 h-6 text-primary" />
            <span className="group-data-[collapsible=icon]:hidden">PromptMan</span>
          </Link>
        </SidebarHeader>
        <Separator className="bg-sidebar-border" />
        <SidebarContent className="py-2">
          <SidebarMenu>
            {NAV_ITEMS.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton
                  render={<Link to={item.url} />}
                  isActive={location.pathname === item.url}
                  tooltip={item.title}
                >
                  <item.icon />
                  <span>{item.title}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarContent>
        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton render={<Link to="/settings" />} tooltip="Settings">
                <Settings />
                <span>Settings</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
          <div className="px-3 py-2 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
            v0.1.0-alpha
          </div>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4 sticky top-0 bg-background z-10">
          <SidebarTrigger />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <div className="flex-1">
            <h2 className="text-sm font-semibold truncate">{currentTitle}</h2>
          </div>
          <div className="flex items-center gap-3">
            <EnvSwitcher />
            <Separator orientation="vertical" className="h-4" />
            <ConnectionStatus />
          </div>
        </header>
        <div className="flex flex-1 flex-col overflow-auto">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
