import { get, put, del } from '@/utils/request'

export interface SharedMcpConfig {
  tenant_id: number
  enabled: boolean
  tools: string[]
  knowledge_base_ids: string[]
  default_agent_id: string
  rate_limit_per_minute: number
  redirect_uri_allowlist: string[]
}

interface Envelope<T> {
  success: boolean
  data: T
  message?: string
}

export interface SharedMcpGrant {
  id: string
  client_id: string
  tenant_id: number
  created_at: string
  revoked_at?: string
}

export function getSharedMcpConfig() {
  return get<Envelope<SharedMcpConfig>>('/api/v1/shared-mcp')
}

export function updateSharedMcpConfig(data: Partial<SharedMcpConfig> & { enabled?: boolean }) {
  return put<Envelope<SharedMcpConfig>>('/api/v1/shared-mcp', data)
}

export function listSharedMcpGrants() {
  return get<Envelope<SharedMcpGrant[]>>('/api/v1/oauth/mcp/grants')
}

export function revokeSharedMcpGrant(grantId: string) {
  return del<Envelope<{ id: string; revoked: boolean }>>(`/api/v1/oauth/mcp/grants/${grantId}`)
}
