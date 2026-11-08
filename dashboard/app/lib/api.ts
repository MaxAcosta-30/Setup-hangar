// Cliente de API - todas las llamadas van a /api/v1/* que Next.js
// proxea a http://localhost:3000/api/v1/* (ver next.config.js)

// HANGAR_API_URL permite correr la API en otro puerto; por defecto, el de siempre.
const API_URL = process.env.HANGAR_API_URL ?? 'http://localhost:3000'
const BASE = typeof window === 'undefined' ? `${API_URL}/api/v1` : '/api/v1'

export interface App {
  id: string
  name: string
  git_url: string
  subdomain: string
  status: 'idle' | 'building' | 'running' | 'failed' | 'stopped'
  created_at: string
  updated_at: string
}

export interface Deployment {
  id: string
  app_id: string
  commit_sha: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'canceled'
  created_at: string
  updated_at: string
}

export interface DeployLog {
  id: number
  deployment_id: string
  message: string
  logged_at: string
}

// Apps
export async function getApps(): Promise<App[]> {
  const res = await fetch(`${BASE}/apps`, { cache: 'no-store' })
  const data = await res.json()
  return data.apps ?? []
}

export async function getApp(id: string): Promise<App> {
  const res = await fetch(`${BASE}/apps/${id}`, { cache: 'no-store' })
  return res.json()
}

export async function createApp(name: string, gitUrl: string) {
  const res = await fetch(`${BASE}/apps`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, git_url: gitUrl }),
  })
  return res.json()
}

// Deployments
export async function getDeployments(appId: string): Promise<Deployment[]> {
  const res = await fetch(`${BASE}/apps/${appId}/deployments`, { cache: 'no-store' })
  const data = await res.json()
  return data.deployments ?? []
}

export async function triggerDeploy(appId: string) {
  const res = await fetch(`${BASE}/apps/${appId}/deploy`, { method: 'POST' })
  return res.json()
}

export async function getLogs(deploymentId: string): Promise<DeployLog[]> {
  const res = await fetch(`${BASE}/deployments/${deploymentId}/logs`, { cache: 'no-store' })
  const data = await res.json()
  return data.logs ?? []
}
