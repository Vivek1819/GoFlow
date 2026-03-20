import { Handle, Position } from "reactflow";
import { CheckCircle, XCircle, Loader } from "lucide-react";

type Props = {
  data: {
    label: string;
    type: string;
    status?: "idle" | "running" | "completed" | "failed";
  };
};

export default function BaseNode({ data }: Props) {

  // ✅ Status-based styling
  const getStatusStyles = (status?: string) => {
    switch (status) {
      case "completed":
        return "border-green-500/40 bg-green-500/5 shadow-green-500/10";
      case "failed":
        return "border-red-500/40 bg-red-500/5 shadow-red-500/10";
      case "running":
        return "border-yellow-400/50 bg-yellow-400/5 animate-pulse shadow-yellow-400/20";
      default:
        return "border-zinc-700 bg-[#111114]";
    }
  };

  // ✅ Icon
  const getStatusIcon = () => {
    switch (data.status) {
      case "running":
        return <Loader size={14} className="animate-spin text-yellow-400" />;
      case "completed":
        return <CheckCircle size={14} className="text-green-400" />;
      case "failed":
        return <XCircle size={14} className="text-red-400" />;
      default:
        return null;
    }
  };

  return (
    <div
      className={`
        relative
        border
        rounded-xl
        px-4 py-3
        min-w-[170px]
        backdrop-blur-md
        transition-all duration-200
        hover:scale-[1.03]
        hover:shadow-lg
        ${getStatusStyles(data.status)}
      `}
    >

      {/* Top row */}
      <div className="flex justify-between items-center mb-2">
        <span className="text-[10px] tracking-wide text-zinc-400 uppercase">
          {data.type}
        </span>
        {getStatusIcon()}
      </div>

      {/* Label */}
      <div className="text-sm font-medium text-white">
        {data.label}
      </div>

      {/* Subtle glow effect (only for running) */}
      {data.status === "running" && (
        <div className="absolute inset-0 rounded-xl border border-yellow-400/20 animate-ping pointer-events-none" />
      )}

      {/* Handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="!bg-zinc-500 !w-2 !h-2"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!bg-zinc-500 !w-2 !h-2"
      />
    </div>
  );
}