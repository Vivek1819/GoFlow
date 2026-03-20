import ReactFlow, {
    Background,
    Controls,
    type Node,
    type Edge,
} from "reactflow";
import BaseNode from "./nodes/BaseNode";
import { useEffect, useState } from "react";
import { fetchWorkflow } from "../../api/workflows";

const nodeTypes = {
    base: BaseNode,
};

export default function WorkflowCanvas({ workflowId }: any) {
    const [nodes, setNodes] = useState<Node[]>([]);
    const [edges, setEdges] = useState<Edge[]>([]);

    useEffect(() => {
        if (!workflowId) return;

        fetchWorkflow(workflowId).then((wf) => {
            const steps =
                typeof wf.steps === "string" ? JSON.parse(wf.steps) : wf.steps;

            if (!steps || !Array.isArray(steps)) return;

            let newNodes: Node[] = [];
            let newEdges: Edge[] = [];

            let x = 100;

            steps.forEach((step: any, index: number) => {
                // ✅ Create main step node
                newNodes.push({
                    id: step.id,
                    type: "base",
                    position: { x: x, y: 150 },
                    data: {
                        label: step.id,
                        type: step.type,
                        status: wf.status === "failed" ? "failed" : "completed",
                    },
                });

                // ✅ Connect previous → current
                if (index > 0 && steps[index - 1].type !== "parallel") {
                    newEdges.push({
                        id: `e-${steps[index - 1].id}-${step.id}`,
                        source: steps[index - 1].id,
                        target: step.id,
                    });
                }

                // ✅ PARALLEL handling (fan-out + fan-in)
                if (step.type === "parallel") {
                    const nextStep = steps[index + 1];

                    step.branches?.forEach((branch: any, i: number) => {
                        // create branch node
                        newNodes.push({
                            id: branch.id,
                            type: "base",
                            position: {
                                x: x + 250,
                                y: 50 + i * 140,
                            },
                            data: {
                                label: branch.id,
                                type: branch.type,
                                status: branch.id.includes("fail")
                                    ? "failed"
                                    : "completed",
                            },
                        });

                        // fan-out: parallel → branch
                        newEdges.push({
                            id: `e-${step.id}-${branch.id}`,
                            source: step.id,
                            target: branch.id,
                        });

                        // 🔥 fan-in: branch → next step
                        if (nextStep) {
                            newEdges.push({
                                id: `e-${branch.id}-${nextStep.id}`,
                                source: branch.id,
                                target: nextStep.id,
                            });
                        }
                    });

                    x += 500;
                } else {
                    x += 250;
                }
            });

            setNodes(newNodes);
            setEdges(newEdges);
        });
    }, [workflowId]);

    return (
        <div className="flex-1 h-full">
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