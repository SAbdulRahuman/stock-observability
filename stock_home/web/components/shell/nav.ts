import {
  Bell,
  BellRing,
  ChartLine,
  ChartNoAxesCombined,
  Flame,
  Gauge,
  Grid2x2,
  Heart,
  House,
  LayoutDashboard,
  type LucideIcon,
  Search,
  Settings,
  TrendingUp,
} from "lucide-react";

export type NavItem = {
  label: string;
  icon: LucideIcon;
  /** In-app route; pages not built yet stay listed but disabled. */
  href?: string;
  ready?: boolean;
  /** Same-origin link id from /api/v1/links, opened as an external tool. */
  linkId?: string;
  badge?: "alerts";
};

/** The single sidebar definition; pages never add their own nav. */
export const navItems: NavItem[] = [
  { label: "Home", icon: House, href: "/", ready: true },
  { label: "Market Overview", icon: ChartLine, href: "/market", ready: true },
  { label: "Watchlist", icon: Heart, href: "/watchlist", ready: true },
  { label: "Alerts", icon: Bell, href: "/alerts", badge: "alerts", ready: true },
  { label: "Top Movers", icon: TrendingUp, href: "/movers", ready: true },
  { label: "Sector View", icon: Grid2x2, href: "/sectors", ready: true },
  { label: "Stock Explorer", icon: Search, href: "/explorer", ready: true },
  { label: "Dashboards", icon: LayoutDashboard, href: "/dashboards", ready: true },
  { label: "OpenBB Analysis", icon: ChartNoAxesCombined, href: "/openbb", ready: true },
  { label: "Prometheus", icon: Flame, href: "/observability/prometheus", ready: true },
  { label: "Grafana", icon: Gauge, linkId: "grafana" },
  { label: "AlertManager", icon: BellRing, linkId: "alertmanager" },
  { label: "Settings", icon: Settings, href: "/settings" },
];

export function navItem(label: string): NavItem | undefined {
  return navItems.find((n) => n.label === label);
}

export function isActive(item: NavItem, pathname: string): boolean {
  if (!item.href) return false;
  return item.href === "/" ? pathname === "/" : pathname === item.href || pathname.startsWith(`${item.href}/`);
}
