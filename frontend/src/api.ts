const API_BASE_URL = import.meta.env.REACT_APP_API_URL || 'http://localhost:8082';

export interface ImportStatus {
  isLoading: boolean;
  currentStep: string;
  progress: number;
  lastUpdated: string;
  error: string;
  totalTitles: number;
  currentTitle: number;
  overallStep: number;
  totalSteps: number;
  agenciesDone: boolean;
  titlesDone: boolean;
  referencesDone: boolean;
  contentDone: boolean;
  historicalDone: boolean;
}

export interface Agency {
  id: string;
  name: string;
  slug: string;
  wordCount: number;
  percentOfTotal: number;
  titleCount: number;
  checksum?: string;
  parentId?: string;
}

export interface TitleBreakdown {
  titleNumber: number;
  titleName: string;
  wordCount: number;
}

export interface AgencyDetail extends Agency {
  subAgencies: Agency[];
  titleBreakdown: TitleBreakdown[];
}

export interface Title {
  id: string;
  number: number;
  name: string;
  wordCount: number;
  checksum?: string;
  latestAmendedOn?: string;
  upToDateAsOf?: string;
}

export interface WordCountMetrics {
  totalCFRWords: number;
  agencies: Agency[];
}

export interface ChecksumInfo {
  titleNumber: number;
  titleName: string;
  checksum?: string;
  lastChanged: string;
}

export interface AgencyChecksumInfo {
  agencyId: string;
  agencyName: string;
  agencySlug: string;
  checksum?: string;
  lastChanged: string;
}

export interface HistoricalPoint {
  date: string;
  wordCount: number;
  changePercent: number;
}

export interface APIResponse<T> {
  data: T;
  meta: {
    total: number;
    lastUpdated: string;
  };
}

class APIService {
  private async fetchAPI<T>(endpoint: string): Promise<APIResponse<T>> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`);
    if (!response.ok) {
      throw new Error(`API request failed: ${response.status}`);
    }
    return response.json();
  }

  async getStatus(): Promise<ImportStatus> {
    const response = await fetch(`${API_BASE_URL}/api/v1/status`);
    if (!response.ok) {
      throw new Error(`Status request failed: ${response.status}`);
    }
    return response.json();
  }

  async getAgencies(): Promise<APIResponse<Agency[]>> {
    return this.fetchAPI<Agency[]>('/api/v1/agencies');
  }

  async getAgencyDetail(slug: string): Promise<APIResponse<AgencyDetail>> {
    return this.fetchAPI<AgencyDetail>(`/api/v1/agencies/${slug}`);
  }

  async getTitles(): Promise<APIResponse<Title[]>> {
    return this.fetchAPI<Title[]>('/api/v1/titles');
  }

  async getWordCountMetrics(): Promise<APIResponse<WordCountMetrics>> {
    return this.fetchAPI<WordCountMetrics>('/api/v1/metrics/word-counts');
  }

  async getChecksums(): Promise<APIResponse<ChecksumInfo[]>> {
    return this.fetchAPI<ChecksumInfo[]>('/api/v1/metrics/checksums');
  }

  async getAgencyChecksums(): Promise<APIResponse<AgencyChecksumInfo[]>> {
    return this.fetchAPI<AgencyChecksumInfo[]>('/api/v1/metrics/agency-checksums');
  }

  async getHistory(agencySlug?: string, months: number = 12): Promise<APIResponse<HistoricalPoint[]>> {
    const params = new URLSearchParams();
    if (agencySlug) params.append('agency', agencySlug);
    if (months !== 12) params.append('months', months.toString());
    
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.fetchAPI<HistoricalPoint[]>(`/api/v1/metrics/history${query}`);
  }

  async exportData(type: 'agencies' | 'titles' | 'metrics'): Promise<Blob> {
    const response = await fetch(`${API_BASE_URL}/api/v1/export/${type}`);
    if (!response.ok) {
      throw new Error(`Export request failed: ${response.status}`);
    }
    return response.blob();
  }
}

export const api = new APIService();