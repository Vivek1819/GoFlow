export default function StepInspector({ node }: any) {
  if (!node) {
    return (
      <div className="w-80 bg-zinc-900 border-l border-zinc-800 p-4">
        <p className="text-zinc-500">Select a step</p>
      </div>
    );
  }

  const { label, type, status, response } = node.data;

  return (
    <div className="w-80 bg-zinc-900 border-l border-zinc-800 p-4 flex flex-col gap-4">

      <div>
        <h2 className="text-lg font-semibold text-white">{label}</h2>
        <p className="text-sm text-zinc-400">{type}</p>
      </div>

      <div>
        <span className="text-xs text-zinc-400">Status</span>
        <p
          className={`text-sm font-medium ${
            status === "failed"
              ? "text-red-400"
              : "text-green-400"
          }`}
        >
          {status}
        </p>
      </div>

      <div className="flex-1 overflow-auto">
        <span className="text-xs text-zinc-400">Response</span>
        <pre className="text-xs bg-zinc-800 p-2 rounded mt-2 overflow-auto">
          {JSON.stringify(response, null, 2)}
        </pre>
      </div>
    </div>
  );
}