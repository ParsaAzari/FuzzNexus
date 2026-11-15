import type { blobs, bridge, output, store, wordlists } from '../../wailsjs/go/models';

export type ResultDTO = output.ResultDTO;
export type ScanRequestDTO = bridge.ScanRequestDTO;
export type ResultsPageDTO = bridge.ResultsPageDTO;
export type FilteredResultsPageDTO = bridge.FilteredResultsPageDTO;
export type PayloadDTO = blobs.PayloadDTO;
export type SessionDTO = store.SessionDTO;
export type WordlistDTO = wordlists.WordlistDTO;
export type WordlistPageDTO = wordlists.WordlistPageDTO;
export type WordlistStatsDTO = wordlists.WordlistStatsDTO;
export type CleanResultDTO = wordlists.CleanResultDTO;
export type WordlistHealthDTO = wordlists.WordlistHealthDTO;
export interface FilterRule {
  id: string;
  field: string;
  operator: string;
  value: string;
  value2?: string;
  enabled: boolean;
  layer: string;
}

export interface FilterGroup {
  id: string;
  conjunction: string;
  rules: FilterRule[];
  groups?: FilterGroup[];
  enabled: boolean;
}

export type FilterPresetDTO = store.FilterPresetDTO;

export type FilterField = 'status' | 'length' | 'words' | 'lines' | 'duration' | 'url' | 'host' | 'payload';
export type FilterOperator = '=' | '!=' | '<' | '<=' | '>' | '>=' | 'between' | 'in' | 'contains' | 'not_contains' | 'regex';
export type FilterLayer = 'view' | 'engine' | 'both';

export type ScanState = 'Idle' | 'Running' | 'Paused' | 'Stopping' | 'Completed' | 'Failed' | 'Stopped';

export interface ProgressData {
  session_id: number;
  started_at: string;
  req_count: number;
  req_total: number;
  req_sec: number;
  queue_pos: number;
  queue_total: number;
  error_count: number;
  percentage: number;
}

export interface CommittedEvent {
  latest_id: number;
  count: number;
  ts: number;
}

export interface StateEvent {
  session_id: number;
  state: ScanState;
  error?: string;
}

export type SortField = 'id' | 'status' | 'length' | 'words' | 'lines' | 'duration' | 'url';
export type SortDirection = 'asc' | 'desc';
