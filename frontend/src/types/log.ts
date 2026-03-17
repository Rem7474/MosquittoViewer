export interface LogEntry {
  id: number
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
  search: string
  clientId: string
  topic: string
}
