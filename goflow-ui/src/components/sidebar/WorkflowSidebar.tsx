import { Workflow } from "lucide-react";

export default function WorkflowSidebar() {
  return (
    <div className="w-64 border-r border-gray-800 bg-[#0f0f12] flex flex-col">

      <div className="p-4 text-xs text-gray-500 uppercase tracking-wide">
        Workflows
      </div>

      <div className="flex flex-col gap-1 px-2">

        <button className="flex items-center gap-2 px-3 py-2 rounded-md hover:bg-gray-800 transition">
          <Workflow size={16}/>
          Workflow #1
        </button>

        <button className="flex items-center gap-2 px-3 py-2 rounded-md hover:bg-gray-800 transition">
          <Workflow size={16}/>
          Workflow #2
        </button>

      </div>
    </div>
  );
}