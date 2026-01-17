'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Home,
  Star,
  Settings,
  HelpCircle,
  ChevronRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';

const navItems = [
  { icon: Home, label: 'Home', href: '/' },
  { icon: Star, label: 'Watchlist', href: '/watchlist' },
];

const bottomItems = [
  { icon: HelpCircle, label: 'Help', href: '/help' },
  { icon: Settings, label: 'Settings', href: '/settings' },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex w-[72px] flex-col border-r border-border bg-background">
      {/* Logo */}
      <div className="flex h-14 items-center justify-center border-b border-border">
        <span className="text-xl font-bold text-primary">A</span>
      </div>

      {/* Main Navigation */}
      <nav className="flex flex-1 flex-col items-center gap-1 py-4">
        {navItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex flex-col items-center gap-1 rounded-lg px-3 py-2.5 text-[10px] transition-colors',
                isActive
                  ? 'text-foreground bg-accent'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground'
              )}
            >
              <item.icon className="h-5 w-5" />
              <span className="font-medium">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* Bottom Section */}
      <div className="flex flex-col items-center gap-1 border-t border-border py-4">
        {/* Bottom Nav Items */}
        {bottomItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex flex-col items-center gap-1 rounded-lg px-3 py-2 text-[11px] transition-colors',
                isActive
                  ? 'text-foreground bg-accent'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground'
              )}
            >
              <item.icon className="h-5 w-5" />
              <span>{item.label}</span>
            </Link>
          );
        })}

        {/* Collapse Toggle */}
        <button className="mt-2 rounded-lg p-2 text-muted-foreground hover:bg-accent">
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </aside>
  );
}
