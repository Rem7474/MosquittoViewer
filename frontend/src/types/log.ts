export interface LogEntry {
  id: number
  source: string
  timestamp: string
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG'
  message: string
  client_id?: string
  topic?: string
  plugin?: string
  raw: string
}

export interface LogFilters {
  level: 'ALL' | LogEntry['level']
  source: string   // '' = all sources
  search: string
  clientId: string
  topic: string
}

export interface LogSource {
  name: string
  path: string
}
