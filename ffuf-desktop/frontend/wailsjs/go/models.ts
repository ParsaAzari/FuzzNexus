export namespace blobs {
	
	export class PayloadDTO {
	    session_id: number;
	    result_id: number;
	    request_text: string;
	    response_text: string;
	    truncated: boolean;
	    full_size_bytes: number;
	    content_type: string;
	
	    static createFrom(source: any = {}) {
	        return new PayloadDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.result_id = source["result_id"];
	        this.request_text = source["request_text"];
	        this.response_text = source["response_text"];
	        this.truncated = source["truncated"];
	        this.full_size_bytes = source["full_size_bytes"];
	        this.content_type = source["content_type"];
	    }
	}

}

export namespace bridge {
	
	export class FilteredResultsPageDTO {
	    results: output.ResultDTO[];
	    total_matches: number;
	
	    static createFrom(source: any = {}) {
	        return new FilteredResultsPageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], output.ResultDTO);
	        this.total_matches = source["total_matches"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResultsPageDTO {
	    results: output.ResultDTO[];
	    has_more: boolean;
	    last_id: number;
	
	    static createFrom(source: any = {}) {
	        return new ResultsPageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], output.ResultDTO);
	        this.has_more = source["has_more"];
	        this.last_id = source["last_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanRequestDTO {
	    url: string;
	    method: string;
	    headers: Record<string, string>;
	    data: string;
	    wordlist: string;
	    threads: number;
	    rate: number;
	    matchers: string;
	    filters: string;
	    timeout: number;
	    follow_redirects: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanRequestDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.data = source["data"];
	        this.wordlist = source["wordlist"];
	        this.threads = source["threads"];
	        this.rate = source["rate"];
	        this.matchers = source["matchers"];
	        this.filters = source["filters"];
	        this.timeout = source["timeout"];
	        this.follow_redirects = source["follow_redirects"];
	    }
	}

}

export namespace output {
	
	export class ResultDTO {
	    id: number;
	    pos: number;
	    inputs: Record<string, string>;
	    status: number;
	    length: number;
	    words: number;
	    lines: number;
	    duration_ms: number;
	    url: string;
	    redirect_location?: string;
	    content_type?: string;
	    hash?: string;
	    host: string;
	    scraper_data?: Record<string, string>;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pos = source["pos"];
	        this.inputs = source["inputs"];
	        this.status = source["status"];
	        this.length = source["length"];
	        this.words = source["words"];
	        this.lines = source["lines"];
	        this.duration_ms = source["duration_ms"];
	        this.url = source["url"];
	        this.redirect_location = source["redirect_location"];
	        this.content_type = source["content_type"];
	        this.hash = source["hash"];
	        this.host = source["host"];
	        this.scraper_data = source["scraper_data"];
	        this.cancelled = source["cancelled"];
	    }
	}

}

export namespace store {
	
	export class FilterRule {
	    id: string;
	    field: string;
	    operator: string;
	    value: string;
	    value2?: string;
	    enabled: boolean;
	    layer: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.field = source["field"];
	        this.operator = source["operator"];
	        this.value = source["value"];
	        this.value2 = source["value2"];
	        this.enabled = source["enabled"];
	        this.layer = source["layer"];
	    }
	}
	export class FilterGroup {
	    id: string;
	    conjunction: string;
	    rules: FilterRule[];
	    groups?: FilterGroup[];
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FilterGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.conjunction = source["conjunction"];
	        this.rules = this.convertValues(source["rules"], FilterRule);
	        this.groups = this.convertValues(source["groups"], FilterGroup);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FilterPresetDTO {
	    id: number;
	    name: string;
	    // Go type: time
	    created_at: any;
	    filter_json: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterPresetDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.filter_json = source["filter_json"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SessionDTO {
	    id: number;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    finished_at?: any;
	    state: string;
	    config_json: string;
	    config_redacted_json: string;
	    total_requests: number;
	    last_error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.finished_at = this.convertValues(source["finished_at"], null);
	        this.state = source["state"];
	        this.config_json = source["config_json"];
	        this.config_redacted_json = source["config_redacted_json"];
	        this.total_requests = source["total_requests"];
	        this.last_error = source["last_error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace wordlists {
	
	export class CleanResultDTO {
	    id: number;
	    lines_before: number;
	    lines_after: number;
	    empty_lines_removed: number;
	    lines_trimmed: number;
	
	    static createFrom(source: any = {}) {
	        return new CleanResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.lines_before = source["lines_before"];
	        this.lines_after = source["lines_after"];
	        this.empty_lines_removed = source["empty_lines_removed"];
	        this.lines_trimmed = source["lines_trimmed"];
	    }
	}
	export class WordlistDTO {
	    id: number;
	    name: string;
	    path: string;
	    line_count: number;
	    size_bytes: number;
	    is_linked: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new WordlistDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.line_count = source["line_count"];
	        this.size_bytes = source["size_bytes"];
	        this.is_linked = source["is_linked"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WordlistHealthDTO {
	    id: number;
	    total_lines: number;
	    blank_lines_count: number;
	    has_whitespace: boolean;
	    has_crlf: boolean;
	    is_clean: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WordlistHealthDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.total_lines = source["total_lines"];
	        this.blank_lines_count = source["blank_lines_count"];
	        this.has_whitespace = source["has_whitespace"];
	        this.has_crlf = source["has_crlf"];
	        this.is_clean = source["is_clean"];
	    }
	}
	export class WordlistPageDTO {
	    lines: string[];
	    total_lines: number;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new WordlistPageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.total_lines = source["total_lines"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	    }
	}
	export class WordlistStatsDTO {
	    line_count: number;
	    size_bytes: number;
	    unique_lines: number;
	    avg_line_len: number;
	    max_line_len: number;
	
	    static createFrom(source: any = {}) {
	        return new WordlistStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line_count = source["line_count"];
	        this.size_bytes = source["size_bytes"];
	        this.unique_lines = source["unique_lines"];
	        this.avg_line_len = source["avg_line_len"];
	        this.max_line_len = source["max_line_len"];
	    }
	}

}

