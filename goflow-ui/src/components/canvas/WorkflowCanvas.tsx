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
import { runWorkflow } from "../../api/workflows";
import { cancelWorkflow } from "../../api/workflows";

const nodeTypes = {
    base: BaseNode,
};

export default function WorkflowCanvas({ workflowId }: any) {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const [selectedNode, setSelectedNode] = useState<any>(null);
    const [stepRuns, setStepRuns] = useState<any[]>([]);
    const [isRunning, setIsRunning] = useState(false);
    const [isCancelling, setIsCancelling] = useState(false);

    useEffect(() => {
        const handleKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                setSelectedNode(null);
            }
        };

        window.addEventListener("keydown", handleKey);
        return () => window.removeEventListener("keydown", handleKey);
    }, []);

    useEffect(() => {
        if (!workflowId) return;

        let interval: any;

        const load = async () => {
            const wf = await fetchWorkflow(workflowId);
            const runs = await fetchWorkflowSteps(workflowId);

            setStepRuns(runs);

            const steps =
                typeof wf.steps === "string" ? JSON.parse(wf.steps) : wf.steps;

            if (!steps || !Array.isArray(steps)) return;

            // 🔥 Build status map
            const stepStatusMap: Record<string, string> = {};
            runs.forEach((s: any) => {
                stepStatusMap[s.step_id] = s.status;
            });

            let newNodes: Node[] = [];
            let newEdges: Edge[] = [];

            let x = 100;

            steps.forEach((step: any, index: number) => {

                const stepContext = wf.context?.[step.id];
                const status = stepStatusMap[step.id] || "pending";

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
                        animated: status === "running",
                        style: {
                            stroke:
                                status === "running"
                                    ? "#facc15"
                                    : status === "completed"
                                        ? "#22c55e"
                                        : status === "failed"
                                            ? "#ef4444"
                                            : status === "cancelled"
                                                ? "#94a3b8" // gray
                                                : "#64748b",
                        }
                    });
                }

                if (step.type === "parallel") {
                    const nextStep = steps[index + 1];

                    step.branches?.forEach((branch: any, i: number) => {

                        const branchContext = wf.context?.[branch.id];
                        const branchStatus = stepStatusMap[branch.id] || "pending";

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
                            animated: branchStatus === "running",
                        });

                        if (nextStep) {
                            newEdges.push({
                                id: `e-${branch.id}-${nextStep.id}`,
                                source: branch.id,
                                target: nextStep.id,
                                animated: branchStatus === "running",
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
        };

        load();

        interval = setInterval(load, 1000);

        return () => clearInterval(interval);

    }, [workflowId]);

    return (
        <div className="flex h-full relative">

            {/* LEFT: Canvas */}
            <div className="flex-1 relative">
                <div className="absolute top-4 right-4 z-10 flex gap-2">

                    <button
                        onClick={async () => {
                            if (!workflowId || isRunning) return;

                            try {
                                setIsRunning(true);

                                await runWorkflow(workflowId);

                                // 🔥 force immediate refresh instead of waiting 1s
                                const wf = await fetchWorkflow(workflowId);
                                const runs = await fetchWorkflowSteps(workflowId);
                                setStepRuns(runs);

                            } catch (err) {
                                console.error("Run failed:", err);
                            } finally {
                                setIsRunning(false);
                            }
                        }}
                        className={`
                            group relative flex items-center gap-2
                            px-5 py-2.5
                            text-sm font-medium
                            rounded-xl
                            backdrop-blur-xl
                            transition-all duration-300

                            ${isRunning
                                ? "bg-emerald-500/20 text-emerald-300 border-emerald-500/40 cursor-not-allowed"
                                : "text-gray-200 bg-[#111114]/80 border-white/10 hover:bg-[#1a1a1f] hover:border-emerald-500/50 hover:shadow-[0_0_20px_rgba(16,185,129,0.2)] hover:-translate-y-0.5"}
                    `}
                    >
                        {/* Glow effect on hover */}
                        <div className="absolute inset-0 bg-gradient-to-r from-emerald-500/10 to-teal-500/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

                        <svg
                            className="w-4 h-4 text-emerald-500 fill-current group-hover:text-emerald-400 group-hover:scale-110 drop-shadow-[0_0_8px_rgba(16,185,129,0.5)] transition-all duration-300 relative z-10"
                            viewBox="0 0 24 24"
                        >
                            <path d="M8 5v14l11-7z" />
                        </svg>
                        <span className="relative z-10 group-hover:text-white transition-colors duration-300">{isRunning ? "Running..." : "Run Workflow"}</span>
                    </button>

                    <button
                        onClick={async () => {
                            if (!workflowId || isCancelling) return;

                            try {
                                setIsCancelling(true);

                                await cancelWorkflow(workflowId);

                            } catch (err) {
                                console.error("Cancel failed:", err);
                            } finally {
                                setIsCancelling(false);
                            }
                        }}
                        disabled={isCancelling}
                        className={`
                            group relative flex items-center gap-2
                            px-5 py-2.5
                            text-sm font-medium
                            rounded-xl
                            backdrop-blur-xl
                            transition-all duration-300

                            ${isCancelling
                                ? "bg-red-500/20 text-red-300 border-red-500/40 cursor-not-allowed"
                                : "text-gray-200 bg-[#111114]/80 border border-white/10 hover:bg-[#1a1a1f] hover:border-red-500/50 hover:shadow-[0_0_20px_rgba(239,68,68,0.2)] hover:-translate-y-0.5"}
                        `}
                    >
                        {/* Icon */}
                        {isCancelling ? (
                            <div className="w-4 h-4 border-2 border-red-400 border-t-transparent rounded-full animate-spin" />
                        ) : (
                            <svg
                                className="w-4 h-4 text-red-500 group-hover:text-red-400 transition-all duration-300"
                                viewBox="0 0 24 24"
                                fill="currentColor"
                            >
                                <path d="M6 6h12v12H6z" />
                            </svg>
                        )}

                        <span>
                            {isCancelling ? "Cancelling..." : "Cancel"}
                        </span>
                    </button>

                </div>
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

            {selectedNode && (
                <div
                    className="absolute inset-0 z-40"
                    onClick={() => setSelectedNode(null)}
                >
                    <div onClick={(e) => e.stopPropagation()}>
                        <StepInspector
                            node={selectedNode}
                            onClose={() => setSelectedNode(null)}
                        />
                    </div>
                </div>
            )}

        </div>
    );
}