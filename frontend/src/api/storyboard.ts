import client from './client'
import type {
  StoryboardProject,
  StoryboardShot,
  CreateProjectRequest,
  UpdateProjectRequest,
  CreateShotRequest,
  UpdateShotRequest,
} from '../types'

export async function listProjects(): Promise<StoryboardProject[]> {
  const res = await client.get('/api/v1/storyboard/projects')
  return res.data.projects
}

export async function getProject(id: number): Promise<{ project: StoryboardProject; shots: StoryboardShot[] }> {
  const res = await client.get(`/api/v1/storyboard/projects/${id}`)
  return res.data
}

export async function createProject(data: CreateProjectRequest): Promise<number> {
  const res = await client.post('/api/v1/storyboard/projects', data)
  return res.data.id
}

export async function updateProject(id: number, data: UpdateProjectRequest): Promise<void> {
  await client.put(`/api/v1/storyboard/projects/${id}`, data)
}

export async function deleteProject(id: number): Promise<void> {
  await client.delete(`/api/v1/storyboard/projects/${id}`)
}

export async function duplicateProject(id: number): Promise<number> {
  const res = await client.post(`/api/v1/storyboard/projects/${id}/duplicate`)
  return res.data.id
}

export async function createShot(projectId: number, data: CreateShotRequest): Promise<number> {
  const res = await client.post(`/api/v1/storyboard/projects/${projectId}/shots`, data)
  return res.data.id
}

export async function updateShot(id: number, data: UpdateShotRequest): Promise<void> {
  await client.put(`/api/v1/storyboard/shots/${id}`, data)
}

export async function deleteShot(id: number): Promise<void> {
  await client.delete(`/api/v1/storyboard/shots/${id}`)
}

export async function reorderShots(projectId: number, ids: number[]): Promise<void> {
  await client.put(`/api/v1/storyboard/projects/${projectId}/shots/reorder`, { ids })
}

export async function generateShots(projectId: number): Promise<void> {
  await client.post(`/api/v1/storyboard/projects/${projectId}/generate`)
}
