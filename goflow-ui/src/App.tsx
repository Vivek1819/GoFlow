import MainLayout from "./layouts/MainLayout";
import WorkflowSidebar from "./components/sidebar/WorkflowSidebar";
import WorkflowCanvas from "./components/canvas/WorkflowCanvas";
import ExecutionPanel from "./components/execution/ExecutionPanel";

export default function App() {
  return (
    <MainLayout>

      <WorkflowSidebar />

      <div className="flex flex-col flex-1">

        <WorkflowCanvas />

        <ExecutionPanel />

      </div>

    </MainLayout>
  );
}