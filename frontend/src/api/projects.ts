import client from './client'
import type { Project, CreateProjectRequest, UpdateProjectRequest, ProjectStepRequest, ProjectStep, ProjectFile, ProjectStats } from '../types'

export async function listProjects(): Promise<Project[]> {
  const res = await client.get('/api/v1/projects')
  return res.data.projects
}

export async function getProject(id: number): Promise<Project> {
  const res = await client.get(`/api/v1/projects/${id}`)
  return res.data.project
}

export async function createProject(data: CreateProjectRequest): Promise<Project> {
  const res = await client.post('/api/v1/projects', {
    title: data.title,
    brief: data.brief,
    type: data.type || 'project',
  })
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

// ==================== 项目仪表盘 ====================

export async function getProjectFiles(id: number): Promise<ProjectFile[]> {
  const res = await client.get(`/api/v1/projects/${id}/files`)
  return res.data.files
}

export async function getProjectStats(id: number): Promise<ProjectStats> {
  const res = await client.get(`/api/v1/projects/${id}/stats`)
  return res.data
}

export async function updateStepProgress(id: number, step: string, status: string): Promise<void> {
  await client.put(`/api/v1/projects/${id}/step-progress`, { step, status })
}
