import { useEffect, useState } from "react";
import { Workflow } from "lucide-react";
import { fetchWorkflows } from "../../api/workflows";

export default function WorkflowSidebar({ onSelect, selectedId }: any) {

    const [workflows, setWorkflows] = useState([]);

    useEffect(() => {
        console.log("FETCHING WORKFLOWS");

        fetchWorkflows().then((data) => {
            console.log("WORKFLOWS:", data);
            setWorkflows(data);
        });
    }, []);

    if (!workflows.length) {
        return (
            <div className="w-64 border-r border-gray-800 flex items-center justify-center text-gray-500 text-sm">
                No workflows yet
            </div>
        );
    }

    return (
        <div className="w-64 border-r border-gray-800 bg-[#0f0f12] flex flex-col">

            <div className="p-4 text-xs text-gray-500 uppercase tracking-wide">
                Workflows
            </div>

            <div className="flex flex-col gap-1 px-2">

                {workflows.map((wf: any) => (
                    <button
                        key={wf.id}
                        onClick={() => onSelect(wf.id)}
                        className={`flex items-center gap-2 px-3 py-2 rounded-md transition
                            ${selectedId === wf.id
                                ? "bg-blue-600/20 border border-blue-500/30 text-blue-300"
                                : "hover:bg-gray-800 text-gray-300"
                            }`}
                    >
                        <Workflow size={16} />
                        Workflow #{wf.id}
                    </button>
                ))}

            </div>
        </div>
    );
}