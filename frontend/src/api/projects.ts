import client from './client'
import type { Project, CreateProjectRequest, UpdateProjectRequest, ProjectStepRequest, ProjectStep } from '../types'

export async function listProjects(): Promise<Project[]> {
  const res = await client.get('/api/v1/projects')
  return res.data.projects
}

export async function getProject(id: number): Promise<Project> {
  const res = await client.get(`/api/v1/projects/${id}`)
  return res.data.project
}

export async function createProject(data: CreateProjectRequest): Promise<Project> {
  const res = await client.post('/api/v1/projects', data)
  return res.data.project
}

export async function updateProject(id: number, data: UpdateProjectRequest): Promise<Project> {
  const res = await client.put(`/api/v1/projects/${id}`, data)
  return res.data.project
}

export async function deleteProject(id: number): Promise<void> {
  await client.delete(`/api/v1/projects/${id}`)
}

export async function duplicateProject(id: number): Promise<Project> {
  const res = await client.post(`/api/v1/projects/${id}/duplicate`)
  return res.data.project
}

export async function aiRecommend(id: number): Promise<{ result: string }> {
  const res = await client.post(`/api/v1/projects/${id}/ai-recommend`)
  return res.data
}

export async function addStep(projectId: number, data: ProjectStepRequest): Promise<ProjectStep> {
  const res = await client.post(`/api/v1/projects/${projectId}/steps`, data)
  return res.data.step
}
