import ReactFlow, { Background, Controls } from "reactflow";
import BaseNode from "./nodes/BaseNode";

const nodeTypes = {
  base: BaseNode,
};

const nodes = [
  {
    id: "1",
    type: "base",
    position: { x: 100, y: 100 },
    data: {
      label: "Fetch Data",
      type: "http",
      status: "completed",
    },
  },
  {
    id: "2",
    type: "base",
    position: { x: 350, y: 100 },
    data: {
      label: "Process",
      type: "transform",
      status: "running",
    },
  },
];

const edges = [
  { id: "e1-2", source: "1", target: "2" },
];

export default function WorkflowCanvas() {
  return (
    <div className="flex-1">

      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
      >
        <Background color="#1a1a1a" gap={24} />
        <Controls />
      </ReactFlow>

    </div>
  );
}