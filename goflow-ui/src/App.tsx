import { useState } from "react";
import MainLayout from "./layouts/MainLayout";
import WorkflowSidebar from "./components/sidebar/WorkflowSidebar";
import WorkflowCanvas from "./components/canvas/WorkflowCanvas";
import ExecutionPanel from "./components/execution/ExecutionPanel";

export default function App() {

  const [selectedWorkflow, setSelectedWorkflow] = useState<number | null>(null);

  return (
    <MainLayout>

      <WorkflowSidebar 
        onSelect={setSelectedWorkflow} 
        selectedId={selectedWorkflow}
      />

      <div className="flex flex-col flex-1 h-full">

        <WorkflowCanvas workflowId={selectedWorkflow} />

        <ExecutionPanel />

      </div>

    </MainLayout>
  );
}