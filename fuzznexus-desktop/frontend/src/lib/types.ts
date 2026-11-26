/**
 * Shared TypeScript DTOs and UI filter/scan types aligned with Wails Go bindings.
 */
import type { blobs, bridge, fuzznexus, output, store, wordlists } from '../../wailsjs/go/models';

export type ResultDTO = output.ResultDTO;
export type ScanRequestDTO = bridge.ScanRequestDTO;
export type ParsedRawHTTPRequestDTO = bridge.ParsedRawHTTPRequestDTO;
export type PreflightConfig = fuzznexus.PreflightConfig;
export type VarExtract = fuzznexus.VarExtract;
export type PendingResumeDTO = bridge.PendingResumeDTO;
export type ResultsPageDTO = bridge.ResultsPageDTO;
export type FilteredResultsPageDTO = bridge.FilteredResultsPageDTO;
export type PayloadDTO = blobs.PayloadDTO;
export type SessionDTO = store.SessionDTO;
export type WordlistDTO = wordlists.WordlistDTO;
export type CategoryDTO = wordlists.CategoryDTO;
export type WordlistPageDTO = wordlists.WordlistPageDTO;
export type WordlistStatsDTO = wordlists.WordlistStatsDTO;
export type CleanResultDTO = wordlists.CleanResultDTO;
export type WordlistHealthDTO = wordlists.WordlistHealthDTO;
export type WordlistRiskReportDTO = wordlists.WordlistRiskReportDTO;
export type SanitizeOptions = wordlists.SanitizeOptions;
export type MergeReportDTO = wordlists.MergeReportDTO;
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
  bytes_sent?: number;
  bytes_recv?: number;
  circuit_state?: string;
  circuit_reason?: string;
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

export interface SafeStopPrompt {
  session_id: number;
  title: string;
  message: string;
  ip_hash: string;
  raw_ip?: string;
  reason: string;
  canary_summary?: string;
  buttons: string[];
}

export interface CanaryEvidence {
  root_url: string;
  root_status: number;
  root_waf_header?: string;
  root_error?: string;
  benign_path_url?: string;
  benign_path_status?: number;
  benign_waf_header?: string;
  benign_error?: string;
  checked_at: string;
}

export type AdvisoryVerdict =
  | 'NONE'
  | 'IP_BAN'
  | 'WORDLIST_WAF_TRIGGER'
  | 'RATE_LIMIT'
  | 'ENDPOINT_PROTECTED';

export interface AdvisoryAlert {
  session_id: number;
  verdict: AdvisoryVerdict;
  confidence: number;
  title: string;
  summary: string;
  evidence_header?: string;
  fixed_body_length?: number;
  sample_payloads?: string[];
  canary: CanaryEvidence;
  suggested_actions?: string[];
}

export interface SafeStopTestResult {
  success: boolean;
  new_ip_hash: string;
  new_raw_ip?: string;
  error?: string;
}

export type SortField = 'id' | 'status' | 'length' | 'words' | 'lines' | 'duration' | 'url';
export type SortDirection = 'asc' | 'desc';

export interface NetworkErrorDTO {
  index: number;
  message: string;
  timestamp: string;
}
