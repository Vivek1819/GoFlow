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

  const getStatusIcon = () => {
    switch (data.status) {
      case "running":
        return <Loader size={14} className="animate-spin text-blue-400" />;
      case "completed":
        return <CheckCircle size={14} className="text-green-400" />;
      case "failed":
        return <XCircle size={14} className="text-red-400" />;
      default:
        return null;
    }
  };

  return (
    <div className="bg-[#111114] border border-gray-800 rounded-xl px-4 py-3 min-w-[160px] shadow-lg hover:border-gray-600 transition">

      {/* Top */}
      <div className="flex justify-between items-center mb-2">
        <span className="text-xs text-gray-400 uppercase">
          {data.type}
        </span>
        {getStatusIcon()}
      </div>

      {/* Label */}
      <div className="text-sm font-medium text-white">
        {data.label}
      </div>

      {/* Handles */}
      <Handle type="target" position={Position.Left} className="bg-gray-600" />
      <Handle type="source" position={Position.Right} className="bg-gray-600" />

    </div>
  );
}