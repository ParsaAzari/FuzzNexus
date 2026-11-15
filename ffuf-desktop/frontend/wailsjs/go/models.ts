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

