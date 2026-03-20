import ReactFlow, {
    Background,
    Controls,
    type Node,
    type Edge,
    useNodesState,
    useEdgesState,
} from "reactflow";
import BaseNode from "./nodes/BaseNode";
import { useEffect, useState } from "react";
import { fetchWorkflow, fetchWorkflowSteps } from "../../api/workflows";
import StepInspector from "../workflow/StepInspector";

const nodeTypes = {
    base: BaseNode,
};

export default function WorkflowCanvas({ workflowId }: any) {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const [selectedNode, setSelectedNode] = useState<any>(null);

    useEffect(() => {
        if (!workflowId) return;

        Promise.all([
            fetchWorkflow(workflowId),
            fetchWorkflowSteps(workflowId),
        ]).then(([wf, stepRuns]) => {

            const steps =
                typeof wf.steps === "string" ? JSON.parse(wf.steps) : wf.steps;

            if (!steps || !Array.isArray(steps)) return;

            // 🔥 Build status map
            const stepStatusMap: Record<string, string> = {};

            stepRuns.forEach((s: any) => {
                stepStatusMap[s.step_id] = s.status;
            });

            let newNodes: Node[] = [];
            let newEdges: Edge[] = [];

            let x = 100;

            steps.forEach((step: any, index: number) => {

                const status = stepStatusMap[step.id] || "pending";

                const stepContext = wf.context?.[step.id];

                newNodes.push({
                    id: step.id,
                    type: "base",
                    position: { x: x, y: 150 },
                    data: {
                        label: step.id,
                        type: step.type,
                        status,
                        response: stepContext?.response || null,
                    },
                });

                if (index > 0 && steps[index - 1].type !== "parallel") {
                    newEdges.push({
                        id: `e-${steps[index - 1].id}-${step.id}`,
                        source: steps[index - 1].id,
                        target: step.id,
                    });
                }

                if (step.type === "parallel") {
                    const nextStep = steps[index + 1];

                    step.branches?.forEach((branch: any, i: number) => {

                        const branchStatus =
                            stepStatusMap[branch.id] || "pending";

                        const branchContext = wf.context?.[branch.id];

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
                                status: branchStatus,
                                response: branchContext?.response || null,
                            },
                        });

                        newEdges.push({
                            id: `e-${step.id}-${branch.id}`,
                            source: step.id,
                            target: branch.id,
                        });

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
        <div className="flex h-full">

            {/* LEFT: Canvas */}
            <div className="flex-1">
                <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    nodeTypes={nodeTypes}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    onNodeClick={(_, node) => setSelectedNode(node)}
                    fitView
                >
                    <Background color="#1a1a1a" gap={24} />
                    <Controls />
                </ReactFlow>
            </div>

            {/* RIGHT: Inspector */}
            <StepInspector node={selectedNode} />

        </div>
    );
}