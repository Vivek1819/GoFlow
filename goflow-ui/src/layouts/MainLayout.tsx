import type { ReactNode } from "react";
import { Workflow, Activity } from "lucide-react";

export default function MainLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-screen bg-[#0b0b0d] text-gray-200 flex flex-col">

      {/* Topbar */}
      <header className="h-14 border-b border-gray-800 flex items-center px-6 justify-between">
        <div className="flex items-center gap-3 font-semibold">
          <Workflow size={20} />
          GoFlow
        </div>

        <div className="flex items-center gap-2 text-sm text-gray-400">
          <Activity size={16} />
          Engine Running
        </div>
      </header>

      {/* Body */}
      <div className="flex flex-1 overflow-hidden">
        {children}
      </div>
    </div>
  );
}