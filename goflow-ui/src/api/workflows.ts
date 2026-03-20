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
  const res = await fetch(`http://localhost:8080/workflows/${id}/steps`);
  return res.json();
}