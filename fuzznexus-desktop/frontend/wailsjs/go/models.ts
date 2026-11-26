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
	export class ParsedRawHTTPRequestDTO {
	    method: string;
	    url: string;
	    headers: Record<string, string>;
	    data: string;

	    static createFrom(source: any = {}) {
	        return new ParsedRawHTTPRequestDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	        this.data = source["data"];
	    }
	}
	export class PendingResumeDTO {
	    session_id: number;
	    target_url: string;
	    method: string;
	    wordlist: string;
	    browser_profile: string;
	    tls_profile: string;
	    current_index: number;
	    ip_hash: string;
	    // Go type: time
	    timestamp: any;

	    static createFrom(source: any = {}) {
	        return new PendingResumeDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.target_url = source["target_url"];
	        this.method = source["method"];
	        this.wordlist = source["wordlist"];
	        this.browser_profile = source["browser_profile"];
	        this.tls_profile = source["tls_profile"];
	        this.current_index = source["current_index"];
	        this.ip_hash = source["ip_hash"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	    wordlists?: string[];
	    input_mode?: string;
	    threads: number;
	    rate: number;
	    matchers: string;
	    filters: string;
	    timeout: number;
	    follow_redirects: boolean;
	    auto_calibration: boolean;
	    proxy_url: string;
	    tls_profile: string;
	    browser_profile: string;
	    rotate_on_block: boolean;
	    resume_from_index?: number;
	    replay_proxy_url?: string;
	    raw?: boolean;
	    http2?: boolean;
	    ignore_body?: boolean;
	    close_conn_on_throttle?: boolean;
	    detect_soft_block?: boolean;
	    delay?: string;
	    max_time?: number;
	    max_time_job?: number;
	    stop_on_403?: boolean;
	    stop_on_errors?: boolean;
	    stop_on_all?: boolean;
	    extensions?: string;
	    dirsearch_compat?: boolean;
	    ignore_wordlist_comments?: boolean;
	    encoders?: string[];
	    matcher_mode?: string;
	    matcher_status?: string;
	    matcher_size?: string;
	    matcher_words?: string;
	    matcher_lines?: string;
	    matcher_regexp?: string;
	    matcher_time?: string;
	    filter_mode?: string;
	    filter_status?: string;
	    filter_size?: string;
	    filter_words?: string;
	    filter_lines?: string;
	    filter_regexp?: string;
	    filter_time?: string;
	    auto_calibration_per_host?: boolean;
	    auto_calibration_keyword?: string;
	    auto_calibration_strategies?: string[];
	    auto_calibration_strings?: string[];
	    recursion?: boolean;
	    recursion_depth?: number;
	    recursion_strategy?: string;
	    request_file?: string;
	    request_proto?: string;
	    sni?: string;
	    client_cert?: string;
	    client_key?: string;
	    input_commands?: string[];
	    input_num?: number;
	    input_shell?: string;
	    scrapers?: string;
	    scraper_file?: string;
	    audit_log?: string;
	    debug_log?: string;
	    output_directory?: string;
	    preflights?: fuzznexus.PreflightConfig[];
	    postflights?: fuzznexus.PreflightConfig[];
	    preflight_mode?: string;
	    preflight_error?: string;
	    preflight_anyhost?: boolean;

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
	        this.wordlists = source["wordlists"];
	        this.input_mode = source["input_mode"];
	        this.threads = source["threads"];
	        this.rate = source["rate"];
	        this.matchers = source["matchers"];
	        this.filters = source["filters"];
	        this.timeout = source["timeout"];
	        this.follow_redirects = source["follow_redirects"];
	        this.auto_calibration = source["auto_calibration"];
	        this.proxy_url = source["proxy_url"];
	        this.tls_profile = source["tls_profile"];
	        this.browser_profile = source["browser_profile"];
	        this.rotate_on_block = source["rotate_on_block"];
	        this.resume_from_index = source["resume_from_index"];
	        this.replay_proxy_url = source["replay_proxy_url"];
	        this.raw = source["raw"];
	        this.http2 = source["http2"];
	        this.ignore_body = source["ignore_body"];
	        this.close_conn_on_throttle = source["close_conn_on_throttle"];
	        this.detect_soft_block = source["detect_soft_block"];
	        this.delay = source["delay"];
	        this.max_time = source["max_time"];
	        this.max_time_job = source["max_time_job"];
	        this.stop_on_403 = source["stop_on_403"];
	        this.stop_on_errors = source["stop_on_errors"];
	        this.stop_on_all = source["stop_on_all"];
	        this.extensions = source["extensions"];
	        this.dirsearch_compat = source["dirsearch_compat"];
	        this.ignore_wordlist_comments = source["ignore_wordlist_comments"];
	        this.encoders = source["encoders"];
	        this.matcher_mode = source["matcher_mode"];
	        this.matcher_status = source["matcher_status"];
	        this.matcher_size = source["matcher_size"];
	        this.matcher_words = source["matcher_words"];
	        this.matcher_lines = source["matcher_lines"];
	        this.matcher_regexp = source["matcher_regexp"];
	        this.matcher_time = source["matcher_time"];
	        this.filter_mode = source["filter_mode"];
	        this.filter_status = source["filter_status"];
	        this.filter_size = source["filter_size"];
	        this.filter_words = source["filter_words"];
	        this.filter_lines = source["filter_lines"];
	        this.filter_regexp = source["filter_regexp"];
	        this.filter_time = source["filter_time"];
	        this.auto_calibration_per_host = source["auto_calibration_per_host"];
	        this.auto_calibration_keyword = source["auto_calibration_keyword"];
	        this.auto_calibration_strategies = source["auto_calibration_strategies"];
	        this.auto_calibration_strings = source["auto_calibration_strings"];
	        this.recursion = source["recursion"];
	        this.recursion_depth = source["recursion_depth"];
	        this.recursion_strategy = source["recursion_strategy"];
	        this.request_file = source["request_file"];
	        this.request_proto = source["request_proto"];
	        this.sni = source["sni"];
	        this.client_cert = source["client_cert"];
	        this.client_key = source["client_key"];
	        this.input_commands = source["input_commands"];
	        this.input_num = source["input_num"];
	        this.input_shell = source["input_shell"];
	        this.scrapers = source["scrapers"];
	        this.scraper_file = source["scraper_file"];
	        this.audit_log = source["audit_log"];
	        this.debug_log = source["debug_log"];
	        this.output_directory = source["output_directory"];
	        this.preflights = this.convertValues(source["preflights"], fuzznexus.PreflightConfig);
	        this.postflights = this.convertValues(source["postflights"], fuzznexus.PreflightConfig);
	        this.preflight_mode = source["preflight_mode"];
	        this.preflight_error = source["preflight_error"];
	        this.preflight_anyhost = source["preflight_anyhost"];
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

export namespace fuzznexus {

	export class VarExtract {
	    name: string;
	    regex: string;
	    source?: string;
	    key?: string;

	    static createFrom(source: any = {}) {
	        return new VarExtract(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.regex = source["regex"];
	        this.source = source["source"];
	        this.key = source["key"];
	    }
	}
	export class PreflightConfig {
	    request_file: string;
	    vars: VarExtract[];

	    static createFrom(source: any = {}) {
	        return new PreflightConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request_file = source["request_file"];
	        this.vars = this.convertValues(source["vars"], VarExtract);
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

export namespace scan {

	export class NetworkErrorDTO {
	    index: number;
	    message: string;
	    // Go type: time
	    timestamp: any;

	    static createFrom(source: any = {}) {
	        return new NetworkErrorDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.message = source["message"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	export class SafeStopTestResultDTO {
	    success: boolean;
	    new_ip_hash: string;
	    new_raw_ip?: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new SafeStopTestResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.new_ip_hash = source["new_ip_hash"];
	        this.new_raw_ip = source["new_raw_ip"];
	        this.error = source["error"];
	    }
	}
	export class TargetAnalysis {
	    original: string;
	    normalized: string;
	    scheme: string;
	    host: string;
	    path: string;
	    has_fuzz: boolean;
	    fuzz_location: string;
	    auto_appended: boolean;

	    static createFrom(source: any = {}) {
	        return new TargetAnalysis(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original = source["original"];
	        this.normalized = source["normalized"];
	        this.scheme = source["scheme"];
	        this.host = source["host"];
	        this.path = source["path"];
	        this.has_fuzz = source["has_fuzz"];
	        this.fuzz_location = source["fuzz_location"];
	        this.auto_appended = source["auto_appended"];
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

	export class CategoryDTO {
	    id: string;
	    name: string;
	    color: string;
	    is_default: boolean;

	    static createFrom(source: any = {}) {
	        return new CategoryDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.is_default = source["is_default"];
	    }
	}
	export class CleanResultDTO {
	    id: number;
	    lines_before: number;
	    lines_after: number;
	    empty_lines_removed: number;
	    lines_trimmed: number;
	    duplicates_removed: number;

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
	        this.duplicates_removed = source["duplicates_removed"];
	    }
	}
	export class MergeReportDTO {
	    total_source_lines: number;
	    deduplicated_lines: number;
	    duplicates_removed: number;
	    empty_lines_removed: number;
	    comments_removed: number;
	    traversal_removed: number;
	    lines_trimmed: number;
	    size_bytes: number;

	    static createFrom(source: any = {}) {
	        return new MergeReportDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_source_lines = source["total_source_lines"];
	        this.deduplicated_lines = source["deduplicated_lines"];
	        this.duplicates_removed = source["duplicates_removed"];
	        this.empty_lines_removed = source["empty_lines_removed"];
	        this.comments_removed = source["comments_removed"];
	        this.traversal_removed = source["traversal_removed"];
	        this.lines_trimmed = source["lines_trimmed"];
	        this.size_bytes = source["size_bytes"];
	    }
	}
	export class SanitizeOptions {
	    trim_whitespace: boolean;
	    strip_bom: boolean;
	    remove_empty: boolean;
	    strip_comments: boolean;
	    strip_leading_slash: boolean;
	    strip_traversal_payloads: boolean;
	    case_insensitive: boolean;
	    max_line_len: number;

	    static createFrom(source: any = {}) {
	        return new SanitizeOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trim_whitespace = source["trim_whitespace"];
	        this.strip_bom = source["strip_bom"];
	        this.remove_empty = source["remove_empty"];
	        this.strip_comments = source["strip_comments"];
	        this.strip_leading_slash = source["strip_leading_slash"];
	        this.strip_traversal_payloads = source["strip_traversal_payloads"];
	        this.case_insensitive = source["case_insensitive"];
	        this.max_line_len = source["max_line_len"];
	    }
	}
	export class WordlistDTO {
	    id: number;
	    name: string;
	    path: string;
	    line_count: number;
	    size_bytes: number;
	    is_linked: boolean;
	    category: string;
	    is_builtin: boolean;
	    builtin_id: string;
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
	        this.category = source["category"];
	        this.is_builtin = source["is_builtin"];
	        this.builtin_id = source["builtin_id"];
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
	    duplicate_count: number;
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
	        this.duplicate_count = source["duplicate_count"];
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
	export class WordlistRiskReportDTO {
	    path: string;
	    total_lines: number;
	    traversal_count: number;
	    waf_signature_count: number;
	    sensitive_route_count: number;
	    risk_level: string;
	    sample_risky_lines: string[];
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new WordlistRiskReportDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.total_lines = source["total_lines"];
	        this.traversal_count = source["traversal_count"];
	        this.waf_signature_count = source["waf_signature_count"];
	        this.sensitive_route_count = source["sensitive_route_count"];
	        this.risk_level = source["risk_level"];
	        this.sample_risky_lines = source["sample_risky_lines"];
	        this.summary = source["summary"];
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
