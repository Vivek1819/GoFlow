import axios from "axios";

const API = "http://localhost:8080";

export async function fetchWorkflows() {
  const res = await axios.get(`${API}/workflows`);
  return res.data;
}

export async function fetchWorkflow(id: number) {
  const res = await axios.get(`http://localhost:8080/workflows/${id}`);
  return res.data;
}

export async function fetchWorkflowSteps(id: number) {
  try {
    const res = await fetch(`http://localhost:8080/workflows/${id}/steps`);
    if (!res.ok) {
      console.warn(`Steps API returned ${res.status}:`, await res.text());
      return [];
    }
    return await res.json();
  } catch (err) {
    console.error("Failed to fetch workflow steps:", err);
    return [];
  }
}

export async function runWorkflow(workflowId: number) {
  const res = await fetch(
    `http://localhost:8080/workflows/${workflowId}/run`,
    {
      method: "POST",
    }
  );

  if (!res.ok) {
    throw new Error("Failed to run workflow");
  }

  return res.json();
}

export async function cancelWorkflow(workflowId: number) {
  const res = await fetch(
    `http://localhost:8080/workflows/${workflowId}/cancel`,
    {
      method: "POST",
    }
  );

  if (!res.ok) {
    throw new Error("Failed to cancel workflow");
  }

  return res.json();
}