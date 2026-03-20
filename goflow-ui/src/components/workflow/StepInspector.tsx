import { X } from "lucide-react";

export default function StepInspector({ node, onClose }: any) {
  if (!node) return null;

  const { label, type, status, response } = node.data;

  return (
    <div className="absolute right-6 top-20 w-80 bg-zinc-900/95 backdrop-blur-xl border border-zinc-700 rounded-xl shadow-2xl p-4 z-50">

      {/* Header */}
      <div className="flex justify-between items-start mb-3">

        <div>
          <h2 className="text-white font-semibold">{label}</h2>
          <p className="text-xs text-zinc-400">{type}</p>
        </div>

        {/* ❌ Close button */}
        <button
          onClick={onClose}
          className="text-zinc-400 hover:text-white transition"
        >
          <X size={16} />
        </button>
      </div>

      {/* Status */}
      <div className="mb-3">
        <span
          className={`text-xs px-2 py-1 rounded ${
            status === "failed"
              ? "bg-red-500/20 text-red-400"
              : status === "completed"
              ? "bg-green-500/20 text-green-400"
              : "bg-yellow-500/20 text-yellow-400"
          }`}
        >
          {status}
        </span>
      </div>

      {/* Response */}
      <div className="max-h-64 overflow-auto">
        <pre className="text-xs bg-zinc-800 p-2 rounded">
          {JSON.stringify(response, null, 2)}
        </pre>
      </div>
    </div>
  );
}