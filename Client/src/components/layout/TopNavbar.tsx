import { Link } from "react-router-dom";
import { Search, Book, ChevronDown, User, LogOut, Settings } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface TopNavbarProps {
  onLogout?: () => void;
}

export function TopNavbar({ onLogout }: TopNavbarProps) {
  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-12 bg-surface border-b border-border">
      <div className="flex items-center justify-between h-full px-4">
        {/* Left section - Logo and Project selector */}
        <div className="flex items-center gap-4">
          <Link to="/" className="flex items-center gap-2 text-foreground font-semibold">
            <div className="w-6 h-6 rounded bg-primary flex items-center justify-center">
              <span className="text-xs text-primary-foreground font-bold">λ</span>
            </div>
            <span className="hidden sm:inline">Serverless</span>
          </Link>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 gap-1 text-muted-foreground hover:text-foreground">
                <span className="max-w-[120px] truncate">my-project</span>
                <ChevronDown className="h-3 w-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-48">
              <DropdownMenuItem>my-project</DropdownMenuItem>
              <DropdownMenuItem>production-api</DropdownMenuItem>
              <DropdownMenuItem>staging-env</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem>Create new project</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Center section - Search */}
        <div className="hidden md:flex flex-1 max-w-md mx-4">
          <div className="relative w-full">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              type="search"
              placeholder="Search functions, files..."
              className="h-7 pl-8 bg-background border-border text-sm"
            />
            <kbd className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none hidden sm:inline-flex h-5 select-none items-center gap-1 rounded border border-border bg-muted px-1.5 font-mono text-2xs text-muted-foreground">
              ⌘K
            </kbd>
          </div>
        </div>

        {/* Right section - Docs and Profile */}
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" className="h-7 gap-1.5 text-muted-foreground hover:text-foreground">
            <Book className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Docs</span>
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-7 w-7 rounded-full">
                <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                  <User className="h-3.5 w-3.5 text-primary" />
                </div>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <div className="px-2 py-1.5">
                <p className="text-sm font-medium">John Doe</p>
                <p className="text-xs text-muted-foreground">john@example.com</p>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild>
                <Link to="/settings" className="flex items-center gap-2">
                  <Settings className="h-3.5 w-3.5" />
                  Settings
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onLogout} className="text-destructive">
                <LogOut className="h-3.5 w-3.5 mr-2" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
