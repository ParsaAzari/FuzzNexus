import type { blobs, bridge, output, store } from '../../wailsjs/go/models';

export type ResultDTO = output.ResultDTO;
export type ScanRequestDTO = bridge.ScanRequestDTO;
export type ResultsPageDTO = bridge.ResultsPageDTO;
export type PayloadDTO = blobs.PayloadDTO;
export type SessionDTO = store.SessionDTO;

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
