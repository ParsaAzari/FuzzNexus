#!/usr/bin/env python3
"""
pack_wordlists.py
Generates and packs enterprise-grade, curated wordlists for FuzzNexus Desktop.
Produces both standalone .json in wordlists_packs/ and maximum-compressed .json.gz
in internal/wordlists/data/packs/ for Go embed.FS.
"""

import gzip
import json
import os

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
JSON_OUT_DIR = os.path.join(BASE_DIR, "fuzznexus-desktop", "wordlists_packs")
GZ_OUT_DIR = os.path.join(BASE_DIR, "fuzznexus-desktop", "internal", "wordlists", "data", "packs")

os.makedirs(JSON_OUT_DIR, exist_ok=True)
os.makedirs(GZ_OUT_DIR, exist_ok=True)

def generate_admin_portals():
    roots = [
        "admin", "administrator", "admin-panel", "admin_panel", "adminpanel",
        "controlpanel", "cpanel", "dashboard", "backend", "portal", "manage",
        "manager", "management", "operator", "console", "sysadmin", "siteadmin",
        "superadmin", "super-admin", "master", "root", "adm", "staff", "intern",
        "internal-admin", "backstage", "ops-portal", "corp-auth", "workbench",
        "cockpit", "mission-control", "backoffice", "desk", "staff-login",
        "internal-access", "employee", "corp", "intranet", "infra-admin",
        "support-portal", "panel", "webadmin", "admin-login", "user-admin",
        "memberadmin", "adminarea", "admin_area", "bb-admin", "moderator",
        "admincenter", "admin-zone", "adm_zone", "secure-admin", "admincontrol"
    ]
    subpaths = [
        "", "/login", "/logon", "/signin", "/auth", "/dashboard", "/account",
        "/users", "/config", "/settings", "/system", "/security", "/overview",
        "/index.html", "/index.php", "/home", "/main", "/portal", "/console",
        "/sessions", "/roles", "/permissions", "/audit", "/logs", "/api",
        "/v1", "/v2", "/profile", "/status", "/debug", "/backup", "/edit",
        "/view", "/reports", "/tools", "/setup", "/install", "/update"
    ]
    hex_prefixes = [
        "1a2b", "9f8e", "6a8b", "7c4d", "01", "02", "99", "v1", "v2", "beta",
        "live", "prod", "stage", "test", "dev", "internal", "secret", "private",
        "secure", "cp", "adm", "ops", "corp", "area", "auth", "mgmt", "alpha",
        "core", "main", "root", "hub", "node", "gateway", "service"
    ]
    suffixes = [
        "", ".php", ".html", ".htm", ".aspx", ".asp", ".jsp", ".json", ".action"
    ]

    keywords = set()
    for r in roots:
        for s in subpaths:
            path = f"{r}{s}"
            keywords.add(path)
            for suf in suffixes:
                if suf and not path.endswith(suf) and not s.endswith(".html") and not s.endswith(".php"):
                    keywords.add(f"{path}{suf}")

    for r in ["admin", "panel", "manage", "portal", "dashboard", "adm", "backend", "console"]:
        for h in hex_prefixes:
            keywords.add(f"{r}_{h}")
            keywords.add(f"{r}-{h}")
            keywords.add(f"{h}_{r}")
            keywords.add(f"{h}-{r}")
            keywords.add(f"{r}_{h}/login")
            keywords.add(f"{r}-{h}/login")
            keywords.add(f"{r}_{h}/dashboard")
            keywords.add(f"{r}_{h}/index.php")
            keywords.add(f"{r}_{h}/admin.php")

    # Hash / Base64 realistic variants
    hashes = [
        "21232f297a57a5a743894a0e4a801fc3",  # md5 admin
        "e10adc3949ba59abbe56e057f20f883e",  # md5 123456
        "c3284d0f94606de1fd2af172abb15bf3",  # md5 administrator
        "8c6976e5b5410415bde908bd4dee15df",  # sha256 prefix admin
        "YWRtaW4=",                          # base64 admin
        "YWRtaW5pc3RyYXRvcg==",              # base64 administrator
    ]
    for h in hashes:
        keywords.add(f"admin_{h}")
        keywords.add(f"portal_{h}")
        keywords.add(f"{h}")

    for y in [2022, 2023, 2024, 2025, 2026]:
        keywords.add(f"admin{y}")
        keywords.add(f"admin_{y}")
        keywords.add(f"admin-{y}")
        keywords.add(f"secure-admin-{y}")
        keywords.add(f"portal-{y}")
        keywords.add(f"cpanel-{y}")
        keywords.add(f"backend-{y}")

    return sorted(list(keywords))

def generate_api_modern():
    versions = ["api", "api/v1", "api/v2", "api/v3", "api/v4", "v1", "v2", "v3", "v4", "api/private", "api/internal", "api/public", "api/v1/internal", "api/v1/private"]
    resources = [
        "users", "user", "auth", "login", "register", "token", "tokens", "session",
        "sessions", "accounts", "account", "profile", "admin", "admins", "config",
        "configuration", "settings", "system", "health", "status", "metrics",
        "debug", "logs", "audit", "events", "notifications", "roles", "permissions",
        "tenants", "organizations", "teams", "projects", "files", "uploads",
        "download", "export", "import", "backup", "keys", "secrets", "credentials",
        "payments", "billing", "subscriptions", "invoices", "orders", "checkout",
        "webhooks", "hooks", "integrations", "search", "query", "filter",
        "reports", "analytics", "stats", "telemetry", "internal", "mgmt", "gateway",
        "customers", "products", "items", "categories", "messages", "chats",
        "devices", "locations", "nodes", "clusters", "queues", "jobs", "tasks"
    ]
    actions = ["", "/all", "/list", "/get", "/create", "/update", "/delete", "/me", "/self", "/info", "/detail", "/status", "/search", "/export"]

    keywords = set()
    for v in versions:
        for r in resources:
            for a in actions:
                keywords.add(f"{v}/{r}{a}")

    doc_paths = [
        "swagger", "swagger-ui", "swagger-ui.html", "swagger-ui/index.html",
        "swagger-ui/", "swagger/v1/swagger.json", "swagger.json", "swagger.yaml",
        "api-docs", "api-docs/v1", "v1/api-docs", "v2/api-docs", "v3/api-docs",
        "v3/api-docs/swagger-config", "openapi.json", "openapi.yaml", "openapi",
        "docs", "redoc", "api/docs", "api/redoc", "api/openapi.json", "api/swagger.json",
        "documentation", "api/documentation", "swagger/ui", "api/v1/docs", "doc",
        "api-spec", "spec", "schema", "api/schema", "api/v1/schema", "api-docs/swagger.json"
    ]
    for d in doc_paths:
        keywords.add(d)

    graphql_paths = [
        "graphql", "graphiql", "graphql/console", "api/graphql", "v1/graphql",
        "v2/graphql", "altair", "playground", "graphql/schema", "graphql/query",
        "api/trpc", "trpc", "rpc", "jsonrpc", "api/jsonrpc", "grpc.web", "connect",
        "subscriptions", "ws/graphql", "graphql/v1", "api/rpc", "trpc/auth",
        "trpc/user", "trpc/admin", "graphql/v2", "api/v2/graphql"
    ]
    for g in graphql_paths:
        keywords.add(g)

    actuators = [
        "actuator", "actuator/env", "actuator/health", "actuator/info",
        "actuator/mappings", "actuator/beans", "actuator/metrics", "actuator/prometheus",
        "actuator/heapdump", "actuator/threaddump", "actuator/loggers", "actuator/configprops",
        "actuator/scheduledtasks", "actuator/auditevents", "actuator/gateway/routes",
        "actuator/httptrace", "actuator/caches", "actuator/integrationgraph",
        "actuator/startup", "actuator/busrefresh", "actuator/refresh", "actuator/shutdown",
        "actuator/conditions", "actuator/flyway", "actuator/liquibase", "actuator/sessions",
        "management/actuator", "mgmt/actuator", "api/actuator", "actuator/circuitbreakers",
        "actuator/features", "actuator/quartz", "actuator/sbom"
    ]
    for act in actuators:
        keywords.add(act)

    return sorted(list(keywords))

def generate_frameworks_backend():
    keywords = set()

    # Next.js App Router, Server Actions & React Server Components
    next_items = [
        "_next", "_next/static", "_next/static/chunks", "_next/static/css",
        "_next/static/media", "_next/image", "_next/data", "_rsc", "?_rsc=1",
        "?_rsc=", "?nxtAction=", "?nxtP=", "?actionId=", "actions", "api/actions",
        "api/auth/[...nextauth]", "api/auth/session", "api/auth/providers", "api/auth/csrf",
        "api/auth/signin", "api/auth/signout", "api/auth/callback", "api/revalidate",
        "api/draft", "api/preview", "api/og", "__nextjs_original-stack-frame",
        "_next/image?url=", "api/revalidate?secret=", "api/draft?secret="
    ]
    for item in next_items:
        keywords.add(item)
    for i in range(1, 25):
        keywords.add(f"_next/static/chunks/app/page-{i}.js")
        keywords.add(f"_next/static/chunks/pages/admin-{i}.js")
        keywords.add(f"_next/static/chunks/main-app-{i}.js")

    # Nuxt 3 & Vue
    nuxt_items = [
        "_payload.json", "_payload.js", "_nuxt", "_nuxt/builds/meta", "__nuxt_island",
        "__nuxt_devtools__", "_nuxt/app.manifest.json", "api/_content", "api/_graphql",
        "api/_auth", "_nuxt/entry.js", "_nuxt/manifest.json"
    ]
    for item in nuxt_items:
        keywords.add(item)

    # SvelteKit & Remix
    svelte_remix = [
        "__data.json", "__data.js", "_app/immutable", "__error.html", "_app/version.json",
        "_app/env.js", "?_data=root", "?_data=routes/admin", "?_data=routes/dashboard",
        "__remix_manifest"
    ]
    for item in svelte_remix:
        keywords.add(item)

    # Laravel 10/11 & Symfony
    laravel_sub = [
        "telescope", "telescope/requests", "telescope/commands", "telescope/schedule",
        "telescope/queries", "telescope/models", "telescope/events", "telescope/mail",
        "telescope/gates", "telescope/views", "telescope/redis", "telescope/monitored-tags",
        "horizon", "horizon/api/stats", "horizon/api/workloads", "horizon/api/masters",
        "horizon/api/monitoring", "pulse", "pulse/api", "_ignition/execute-solution",
        "_ignition/health-check", "_ignition/scripts", "livewire/message", "livewire/upload-file",
        "livewire/preview-file", "sanctum/csrf-cookie", "nova", "nova/login", "nova-api",
        "nova-api/users", "_profiler", "_profiler/empty/search/results", "_wdt",
        "storage/logs/laravel.log", "broadcasting/auth", "oauth/clients",
        "oauth/personal-access-tokens", "oauth/tokens"
    ]
    for item in laravel_sub:
        keywords.add(item)

    # Django 5 & Django Ninja / DRF
    django_sub = [
        "__debug__", "__debug__/render_panel", "__debug__/sql_select",
        "__debug__/template_source", "__debug__/sql_explain", "__debug__/history",
        "api/schema/swagger-ui", "api/schema/redoc", "api/schema", "admin/login/?next=/admin/",
        "static/admin", "silk", "silk/requests", "silk/summary", "api-auth/login", "api-auth/logout"
    ]
    for item in django_sub:
        keywords.add(item)

    # Ruby on Rails 7/8
    rails_sub = [
        "sidekiq", "sidekiq/queues", "sidekiq/busy", "sidekiq/retries", "sidekiq/scheduled",
        "sidekiq/dead", "rails/info/properties", "rails/info/routes", "rails/mailers",
        "rails/conductor/action_mailbox/inbound_emails", "rails/active_storage/disk", "rails/health"
    ]
    for item in rails_sub:
        keywords.add(item)

    # Go Runtime pprof & Diagnostics
    go_sub = [
        "debug/pprof", "debug/pprof/cmdline", "debug/pprof/profile", "debug/pprof/symbol",
        "debug/pprof/trace", "debug/pprof/goroutine", "debug/pprof/heap",
        "debug/pprof/threadcreate", "debug/pprof/block", "debug/pprof/mutex",
        "debug/pprof/allocs", "debug/vars"
    ]
    for item in go_sub:
        keywords.add(item)

    # ASP.NET Core
    dotnet_sub = [
        "hangfire", "hangfire/jobs/enqueued", "hangfire/servers", "hangfire/recurring",
        "elmah.axd", "trace.axd", "health", "healthz", "alive", "ready",
        "swagger/v1/swagger.json", "swagger/index.html"
    ]
    for item in dotnet_sub:
        keywords.add(item)

    for fw in ["admin", "api", "v1", "portal", "system"]:
        for ep in ["__data.json", "_payload.json", "?_rsc=1", "health", "metrics", "env"]:
            keywords.add(f"{fw}/{ep}")

    return sorted(list(keywords))

def generate_cms_platforms():
    keywords = set()

    # WordPress Core & JSON API
    wp_core = [
        "wp-login.php", "wp-admin", "wp-admin/admin-ajax.php", "wp-admin/admin-post.php",
        "wp-admin/install.php", "wp-admin/upgrade.php", "wp-admin/setup-config.php",
        "xmlrpc.php", "wp-json", "wp-json/wp/v2/users", "wp-json/wp/v2/posts",
        "wp-json/wp/v2/pages", "wp-json/wp/v2/comments", "wp-json/wp/v2/media",
        "wp-json/oembed/1.0/embed", "wp-content/debug.log", "wp-config.php.bak",
        "wp-config.php.save", "wp-config.php.old", "wp-config.php.swp", "wp-config.php~",
        "wp-content/uploads", "wp-cron.php", "wp-links-opml.php", "wp-mail.php",
        "wp-signup.php", "wp-trackback.php", "readme.html", "license.txt"
    ]
    for p in wp_core:
        keywords.add(p)

    top_plugins = [
        "woocommerce", "elementor", "contact-form-7", "yoast-seo", "wordfence",
        "all-in-one-seo-pack", "jetpack", "wp-file-manager", "updraftplus",
        "duplicator", "wp-super-cache", "w3-total-cache", "revslider", "gravityforms",
        "akismet", "tinymce-advanced", "site-kit-by-google", "really-simple-ssl",
        "advanced-custom-fields", "wpforms-lite", "smush", "wp-optimize"
    ]
    for plug in top_plugins:
        keywords.add(f"wp-content/plugins/{plug}/")
        keywords.add(f"wp-content/plugins/{plug}/readme.txt")
        keywords.add(f"wp-content/plugins/{plug}/changelog.txt")
        keywords.add(f"wp-content/plugins/{plug}/license.txt")

    strapi_paths = [
        "admin", "admin/auth/login", "admin/auth/register", "admin/auth/forgot-password",
        "admin/plugins/users-permissions", "admin/plugins/content-type-builder",
        "admin/plugins/documentation", "admin/plugins/upload", "api/users", "api/users/me",
        "api/upload", "api/upload/files", "documentation", "documentation/v1.0.0/full_documentation.json",
        "api/auth/local", "api/auth/local/register"
    ]
    for p in strapi_paths:
        keywords.add(p)

    headless = [
        "ghost", "ghost/#/signin", "ghost/#/setup", "ghost/api/v4/admin",
        "ghost/api/v3/admin", "ghost/api/content", "ghost/api/v4/content",
        "server/ping", "server/info", "server/health", "auth/login", "auth/refresh",
        "auth/logout", "users/me", "collections", "items", "files", "activity",
        "settings", "roles", "permissions", "presets", "panel", "panel/login"
    ]
    for p in headless:
        keywords.add(p)

    ecom = [
        "admin_1a2b", "backend", "downloader", "setup", "admin-auth", "api/store",
        "shopware", "admin/dashboard", "api/v1/orders", "api/v1/customers",
        "checkout/cart", "rest/all/V1", "rest/V1", "rest/default/V1",
        "oauth/token", "soap/default", "app/etc/local.xml", "var/log/system.log"
    ]
    for p in ecom:
        keywords.add(p)

    drupal_joomla = [
        "user/login", "user/register", "user/password", "core/install.php",
        "core/CHANGELOG.txt", "admin/reports/status", "update.php", "sites/default/settings.php",
        "sites/default/files", "administrator/index.php", "administrator/help",
        "administrator/language", "configuration.php", "configuration.php-dist",
        "jsonapi", "jsonapi/node/article", "jsonapi/user/user", "api/index.php/v1/users"
    ]
    for p in drupal_joomla:
        keywords.add(p)

    # 2026 Headless & Composable CMS (Payload, Directus, Sanity, Contentful, Storyblok, Umbraco)
    modern_headless = [
        "api/users/login", "api/users/me", "api/users/refresh-token", "api/users/forgot-password",
        "api/access", "api/graphql", "api/graphql-playground", "api/media", "api/globals",
        "directus", "admin/login", "server/specs/oas", "server/specs/graphql",
        "items/users", "items/settings", "assets", "extensions", "flows", "webhooks",
        "umbraco", "umbraco/backoffice", "umbraco/api", "umbraco/management/api/v1",
        "studio", "sanity", "v1/data/query/production", "v2021-06-07/data/query/production",
        "spaces", "environments/master/entries", "cdn/spaces", "v1/cdn/stories",
        "wp-json/wp/v2/types", "wp-json/wp/v2/taxonomies", "wp-json/wp/v2/categories",
        "wp-json/wp/v2/tags", "wp-json/wp/v2/search", "wp-json/wp/v2/settings",
        "wp-json/wc/v3/products", "wp-json/wc/v3/orders", "wp-json/wc/v3/customers",
        "wp-json/yoast/v1/get_head", "wp-includes/version.php", "wp-content/uploads/wc-logs"
    ]
    for p in modern_headless:
        keywords.add(p)

    cms_prefixes = ["cms", "blog", "content", "editorial", "store", "shop", "portal"]
    cms_actions = ["admin", "api", "graphql", "health", "login", "uploads", "media", "preview", "draft", "publish", "webhooks", "users", "settings", "export", "import"]
    for pref in cms_prefixes:
        for act in cms_actions:
            keywords.add(f"{pref}/{act}")

    # Extra popular WordPress plugins & themes
    extra_wp = [
        "wp-rocket", "litespeed-cache", "wp-mail-smtp", "wordfence-assistant",
        "redirection", "wp-fastest-cache", "autoptimize", "complianz-gdpr",
        "cookie-notice", "classic-editor", "google-sitemap-generator", "wp-migrate-db",
        "all-in-one-wp-migration", "backwpup", "better-wp-security", "ithemes-security",
        "sucuri-scanner", "wp-cerber", "ninja-forms", "formidable", "fluentform",
        "learndash", "tutor", "buddyboss-platform", "buddypress", "bbpress",
        "wp-job-manager", "give", "easy-digital-downloads", "memberpress",
        "paid-memberships-pro", "polylang", "sitepress-multilingual-cms",
        "shortpixel-image-optimiser", "imagify", "ewww-image-optimizer",
        "wp-graphql", "jwt-authentication-for-wp-rest-api", "disable-xml-rpc",
        "custom-post-type-ui", "pods", "metabox", "jet-engine", "crocoblock",
        "beaver-builder-lite-version", "divi-builder", "brizy", "oxygen",
        "bricks", "spectra", "kadence-blocks", "generateblocks", "stackable",
        "header-footer-elementor", "essential-addons-for-elementor-lite",
        "premium-addons-for-elementor", "royal-elementor-addons", "happy-elementor-addons",
        "unlimited-elements-for-elementor", "woo-checkout-field-editor-pro",
        "yith-woocommerce-wishlist", "woocommerce-payments", "woocommerce-gateway-stripe",
        "klarna-checkout-for-woocommerce", "mailchimp-for-woocommerce", "facebook-for-woocommerce"
    ]
    for plug in extra_wp:
        keywords.add(f"wp-content/plugins/{plug}/")
        keywords.add(f"wp-content/plugins/{plug}/readme.txt")
        keywords.add(f"wp-content/plugins/{plug}/changelog.txt")

    wp_themes = ["astra", "hello-elementor", "generatepress", "oceanwp", "neve", "kadence", "storefront", "twentytwentyfour", "twentytwentyfive", "divi", "avada", "flatsome", "salient", "enfold"]
    for th in wp_themes:
        keywords.add(f"wp-content/themes/{th}/style.css")
        keywords.add(f"wp-content/themes/{th}/readme.txt")

    return sorted(list(keywords))

def generate_cloud_devops():
    keywords = set()

    k8s = [
        "k8s", "api/v1/namespaces", "api/v1/pods", "api/v1/secrets", "api/v1/configmaps",
        "api/v1/services", "apis/apps/v1", "apis/apps/v1/deployments", "apis/apps/v1/daemonsets",
        "apis/apps/v1/statefulsets", "metrics/cadvisor", "healthz", "livez", "readyz",
        "docker/daemon.json", "v2/_catalog", "v2", "v2/tags/list", "version", "openapi/v2"
    ]
    for p in k8s:
        keywords.add(p)

    prom = [
        "prometheus", "metrics", "federate", "graph", "alerts", "rules", "targets",
        "flags", "config", "api/v1/query", "api/v1/query_range", "api/v1/series",
        "api/v1/labels", "api/v1/status/buildinfo", "api/v1/status/runtimeinfo",
        "grafana", "grafana/login", "grafana/api/org", "grafana/api/users",
        "grafana/api/datasources", "grafana/api/dashboards", "grafana/api/admin/settings",
        "kibana", "kibana/app/kibana", "elasticsearch", "elasticsearch/_cat/indices",
        "elasticsearch/_cluster/health", "elasticsearch/_nodes", "fluentd", "jaeger",
        "jaeger/ui", "zipkin", "zipkin/api/v2/spans"
    ]
    for p in prom:
        keywords.add(p)

    # OpenTelemetry & Telemetry Collectors
    otel = [
        "v1/traces", "v1/metrics", "v1/logs", "v1/health", "otlp/v1/traces",
        "otlp/v1/metrics", "otlp/v1/logs", "opentelemetry", "otel/metrics",
        "api/v1/telemetry", "collector/health", "telemetry/v1/metrics"
    ]
    for p in otel:
        keywords.add(p)

    # Dapr Distributed Application Runtime
    dapr = [
        "v1.0/invoke", "v1.0/state", "v1.0/secrets", "v1.0/publish", "v1.0/bindings",
        "v1.0/metadata", "v1.0/healthz", "dapr/config", "dapr/subscribe"
    ]
    for p in dapr:
        keywords.add(p)

    # API Gateways & Service Mesh
    mesh = [
        "envoy/server_info", "envoy/certs", "envoy/clusters", "envoy/listeners",
        "envoy/stats", "envoy/config_dump", "envoy/logging", "traefik/api/overview",
        "traefik/dashboard", "traefik/api/entrypoints", "traefik/api/http/routers",
        "traefik/api/rawdata", "traefik/dashboard/#/", "apisix/admin/routes",
        "apisix/admin/upstreams", "apisix/admin/consumers", "apisix/admin/plugins",
        "kong/services", "kong/routes", "kong/plugins", "kong/status",
        "consul/ui", "v1/agent/self", "v1/status/leader", "v1/kv", "v1/catalog/services",
        "v1/sys/health", "v1/sys/seal-status", "v1/sys/mounts", "v1/auth/token/lookup-self"
    ]
    for p in mesh:
        keywords.add(p)

    meta = [
        "latest/meta-data", "latest/meta-data/iam/security-credentials",
        "latest/dynamic/instance-identity/document", "latest/user-data",
        "computeMetadata/v1", "computeMetadata/v1/instance/service-accounts/default/token",
        "metadata/v1/maintenance", "opc/v1/instance"
    ]
    for p in meta:
        keywords.add(p)

    cicd = [
        "argocd", "argocd/api/v1/session", "drone", "drone/api/user", "jenkins",
        "jenkins/login", "jenkins/script", "jenkins/scriptText", "jenkins/asynchPeople",
        "gitlab", "gitlab/admin", ".gitlab-ci.yml", ".circleci/config.yml",
        ".github/workflows", "teamcity", "bamboo", "sonar", "sonarqube",
        "harbor", "harbor/api/v2.0/projects", "harbor/api/v2.0/systeminfo",
        "portainer", "portainer/api/status", "portainer/api/endpoints",
        "rancher", "v3-public/authProviders", "v1/management.cattle.io.clusters",
        "loki/api/v1/labels", "loki/api/v1/query", "loki/ready",
        "tempo/api/traces", "alertmanager/api/v2/alerts", "alertmanager/api/v2/status",
        "minio/health/live", "minio/health/ready", "minio/login",
        "terraform.tfstate", "terraform.tfstate.backup", "terraform.tfvars",
        "vault/ui", "v1/sys/seal", "v1/auth/userpass/login", "v1/secret/data",
        "nomad/ui", "v1/jobs", "v1/nodes", "v1/allocations",
        "tekton", "argo-workflows", "api/v1/workflows", "buildkite", "woodpecker"
    ]
    for p in cicd:
        keywords.add(p)

    # Extended Cloud, Container & Service Mesh combinations
    cloud_services = [
        "kubernetes", "k8s", "helm", "istio", "linkerd", "cilium", "calico",
        "fluentbit", "vector", "datadog", "newrelic", "sentry", "elastic",
        "opensearch", "redis", "rabbitmq", "kafka", "nats", "etcd", "vault",
        "consul", "nomad", "traefik", "envoy", "haproxy", "nginx_status",
        "stub_status", "server-status", "server-info", "cadvisor", "node-exporter",
        "kube-state-metrics", "blackbox-exporter", "prometheus-pushgateway"
    ]
    cloud_suffixes = ["", "/health", "/healthz", "/ready", "/readyz", "/livez", "/metrics", "/status", "/version", "/info", "/config", "/api", "/ui", "/dashboard", "/stats"]
    for svc in cloud_services:
        for suf in cloud_suffixes:
            keywords.add(f"{svc}{suf}")

    aws_gcp_azure = [
        "latest/meta-data/hostname", "latest/meta-data/local-ipv4", "latest/meta-data/public-ipv4",
        "latest/meta-data/placement/availability-zone", "latest/meta-data/network/interfaces/macs",
        "metadata/instance?api-version=2021-02-01", "metadata/identity/oauth2/token",
        "computeMetadata/v1/project/project-id", "computeMetadata/v1/instance/attributes",
        "100.100.100.200/latest/meta-data", "metadata/v1.json"
    ]
    for p in aws_gcp_azure:
        keywords.add(p)

    return sorted(list(keywords))

def generate_auth_sso():
    keywords = set()

    oidc = [
        ".well-known/openid-configuration", ".well-known/oauth-authorization-server",
        ".well-known/jwks.json", ".well-known/webfinger", ".well-known/assetlinks.json",
        ".well-known/apple-app-site-association", "oauth/v2/token", "oauth/authorize",
        "oauth/token", "oauth/callback", "oauth/keys", "oauth/revoke", "oauth/userinfo",
        "oauth/introspect", "oauth/clients", "auth/realms/master/.well-known/openid-configuration",
        "auth/admin/master/console", "auth/realms/master/protocol/openid-connect/token",
        "auth/realms/master/protocol/openid-connect/auth", "auth/realms/master/protocol/openid-connect/certs"
    ]
    for p in oidc:
        keywords.add(p)

    saml = [
        "saml/sso", "saml/metadata", "saml/consume", "saml/login", "saml/logout",
        "auth/saml/callback", "auth/saml/metadata", "sso/login", "sso/post",
        "sso/callback", "sso/saml", "saml2/idp/SSOService.php"
    ]
    for p in saml:
        keywords.add(p)

    auth = [
        "api/auth/refresh", "api/auth/reset-password", "api/auth/forgot-password",
        "auth/magic-link", "api/v1/auth/mfa/verify", "api/v1/auth/mfa/setup",
        "session/token", "login/otp", "passkey/register", "passkey/authenticate",
        "auth/jwt/refresh", "auth/jwt/verify", "auth/jwt/create", "auth/session",
        "auth/check", "auth/logout", "auth/verify-email", "auth/resend-verification",
        "auth/password/reset", "auth/password/email", "auth/two-factor", "auth/2fa"
    ]
    for p in auth:
        keywords.add(p)

    providers = ["google", "github", "facebook", "apple", "microsoft", "gitlab", "discord", "twitter", "linkedin", "okta", "azure", "slack", "oidc", "saml"]
    for prov in providers:
        keywords.add(f"auth/{prov}")
        keywords.add(f"auth/{prov}/callback")
        keywords.add(f"api/auth/{prov}")
        keywords.add(f"oauth/{prov}")
        keywords.add(f"oauth2/{prov}/callback")
        keywords.add(f"login/{prov}")

    # 2026 Passkey/WebAuthn, OAuth2 Device Flow, IdP & Session Endpoints
    modern_auth = [
        "webauthn/register", "webauthn/register/options", "webauthn/register/verify",
        "webauthn/authenticate", "webauthn/login", "passkey/challenge", "passkey/verify",
        "oauth/device/code", "oauth2/device_authorization", "oauth2/v2.0/authorize",
        "oauth2/v2.0/token", "oauth2/v1/authorize", "oauth2/v1/token", "oauth2/v1/userinfo",
        "oauth2/v1/introspect", "oauth2/v1/revoke", "oauth2/v1/keys",
        "oidc/auth", "oidc/token", "oidc/userinfo", "oidc/jwks", "oidc/logout",
        "saml/acs", "saml/sls", "saml/idp", "saml/sp/metadata", "saml2/acs", "saml2/metadata",
        "api/v1/auth/login", "api/v1/auth/logout", "api/v1/auth/register", "api/v1/auth/refresh",
        "api/v1/auth/me", "api/v1/auth/session", "api/v1/auth/sessions", "api/v1/auth/token",
        "api/v1/auth/magic-link", "api/v1/auth/otp/send", "api/v1/auth/otp/verify",
        "api/v1/auth/mfa/enroll", "api/v1/auth/mfa/challenge", "api/v1/auth/mfa/recovery",
        "api/v1/auth/totp/setup", "api/v1/auth/totp/verify", "api/v1/auth/backup-codes",
        "api/v2/auth/login", "api/v2/auth/token", "api/v2/auth/refresh", "api/v2/auth/session",
        "auth/csrf", "auth/providers", "auth/signin", "auth/signout", "auth/signup",
        "auth/impersonate", "auth/switch-tenant", "auth/whoami", "auth/validate",
        "sso/initiate", "sso/acs", "sso/metadata.xml", "sso/oidc/callback",
        "clerk/v1/client", "supabase/auth/v1/token", "auth/v1/token", "auth/v1/user",
        "auth/v1/recover", "auth/v1/otp", "auth/v1/magiclink", "auth/v1/authorize",
        "identity/connect/token", "identity/connect/authorize", "connect/token", "connect/authorize",
        "connect/userinfo", "connect/introspect", "connect/revocation", ".well-known/uma2-configuration",
        ".well-known/webfinger?resource=acct:admin", "scim/v2/Users", "scim/v2/Groups",
        "scim/v2/ServiceProviderConfig", "api/scim/v2/Users", "api/keys", "api/tokens"
    ]
    for p in modern_auth:
        keywords.add(p)

    auth_prefixes = ["auth", "api/auth", "v1/auth", "v2/auth", "account", "accounts", "identity", "iam", "sso", "oauth", "oauth2", "session", "sessions", "user", "users"]
    auth_actions = ["login", "logout", "signin", "signout", "signup", "register", "token", "refresh", "revoke", "verify", "introspect", "userinfo", "me", "profile", "password", "reset", "forgot", "mfa", "2fa", "otp", "totp", "passkey", "webauthn", "keys", "jwks", "metadata", "callback", "status", "check"]
    for pref in auth_prefixes:
        for act in auth_actions:
            keywords.add(f"{pref}/{act}")

    return sorted(list(keywords))

def generate_leaks_artifacts():
    keywords = set()

    vcs = [
        ".git", ".git/HEAD", ".git/config", ".git/index", ".git/logs/HEAD",
        ".git/COMMIT_EDITMSG", ".git/description", ".git/info/exclude",
        ".gitignore", ".gitmodules", ".gitattributes", ".svn", ".svn/entries",
        ".svn/wc.db", ".hg", ".hg/dirstate", ".bzr"
    ]
    for p in vcs:
        keywords.add(p)

    env_bases = [
        ".env", ".env.local", ".env.production", ".env.development", ".env.stage",
        ".env.test", ".env.backup", ".env.save", ".env.example", ".env.dist",
        ".env.sample", ".env.prod", ".env.dev", ".env.old", ".env.bak"
    ]
    for e in env_bases:
        keywords.add(e)
        keywords.add(f"{e}.bak")
        keywords.add(f"{e}.old")
        keywords.add(f"{e}~")

    configs = [
        "config.json", "config.yaml", "config.yml", "config.js", "config.ts",
        "local_settings.py", "database.yml", "secrets.json", "secrets.yaml",
        "credentials.json", "appsettings.json", "appsettings.Development.json",
        "parameters.yml", "settings.json", "connections.json", "db.config"
    ]
    for c in configs:
        keywords.add(c)
        keywords.add(f"{c}.bak")
        keywords.add(f"{c}.old")

    backup_names = [
        "backup", "bak", "backup_site", "dump", "data", "db", "database",
        "site", "app", "source", "code", "deploy", "release", "archive",
        "export", "files", "sql", "full_backup", "test", "old", "new",
        "www", "htdocs", "public", "master", "prod", "production", "staging"
    ]
    backup_exts = [".zip", ".tar.gz", ".tar", ".gz", ".tgz", ".7z", ".rar", ".sql", ".sqlite", ".db", ".sqlite3"]
    for b in backup_names:
        for ext in backup_exts:
            keywords.add(f"{b}{ext}")
            keywords.add(f"backup/{b}{ext}")
            keywords.add(f"{b}_2024{ext}")
            keywords.add(f"{b}_2025{ext}")
            keywords.add(f"{b}_2026{ext}")

    logs = [
        "storage/logs/laravel.log", "debug.log", "error.log", "app.log",
        "production.log", "development.log", "access.log", "catalina.out",
        "server.log", "output.log", "system.log", "php_errors.log",
        "logs/error.log", "logs/access.log", "logs/app.log"
    ]
    for p in logs:
        keywords.add(p)

    manifests = [
        ".DS_Store", "Thumbs.db", "package.json", "package-lock.json",
        "yarn.lock", "pnpm-lock.yaml", "composer.json", "composer.lock",
        "docker-compose.yml", "docker-compose.override.yml", "Dockerfile",
        "Procfile", "Makefile", "Pipfile", "Pipfile.lock", "requirements.txt",
        "Gemfile", "Gemfile.lock", "pom.xml", "build.gradle", "go.mod", "go.sum",
        "Cargo.toml", "Cargo.lock"
    ]
    for p in manifests:
        keywords.add(p)

    return sorted(list(keywords))

def generate_bypasses_heuristics():
    keywords = set()

    base_targets = ["admin", "api", "v1", "manager", "dashboard", "portal", "system", "internal", "root", "console"]
    for t in base_targets:
        keywords.add(f"{t}/")
        keywords.add(f"{t}//")
        keywords.add(f"{t}/.")
        keywords.add(f"{t}/..;/")
        keywords.add(f"{t}/..;")
        keywords.add(f"{t};")
        keywords.add(f"{t}?")
        keywords.add(f"{t}#")
        keywords.add(f"{t}%20")
        keywords.add(f"{t}%09")
        keywords.add(f".//{t}")
        keywords.add(f"./{t}/.")
        keywords.add(f"/%2e/{t}")
        keywords.add(f"/%252e%252e/{t}")
        keywords.add(f";/{t}")
        keywords.add(f".;/{t}")
        keywords.add(f"{t}.json")
        keywords.add(f"{t}.html")
        keywords.add(f"{t}.xml")
        keywords.add(f"{t}.action")
        keywords.add(f"api/..;/{t}")
        keywords.add(f"v1/../../{t}")
        keywords.add(t.upper())
        keywords.add(t.capitalize())

    params = [
        "?debug=true", "?debug=1", "?test=1", "?test=true", "?admin=1",
        "?admin=true", "?preview=true", "?preview=1", "?show_all=1",
        "?internal=1", "?internal=true", "?format=json", "?format=xml",
        "?env=dev", "?env=development", "?view=admin", "?role=admin",
        "?su=1", "?bypass=1", "?override=1", "?disable_auth=1"
    ]
    for p in params:
        keywords.add(p)
        for t in ["", "index.php", "index.html", "api"]:
            keywords.add(f"{t}{p}")

    return sorted(list(keywords))

def generate_ai_llm():
    keywords = set()

    # 1. Ollama Endpoints
    ollama_endpoints = [
        "api/generate", "api/chat", "api/tags", "api/show", "api/create",
        "api/copy", "api/delete", "api/pull", "api/push", "api/embeddings",
        "api/ps", "api/version", "api/blobs", "ollama", "ollama/api/generate",
        "ollama/api/chat", "ollama/api/tags", "ollama/api/version"
    ]
    for ep in ollama_endpoints:
        keywords.add(ep)

    # 2. OpenAI-Compatible & High-Throughput Engines (vLLM, TGI, LocalAI, LM Studio)
    openai_compatible = [
        "v1/models", "v1/chat/completions", "v1/completions", "v1/embeddings",
        "v1/audio/transcriptions", "v1/audio/translations", "v1/audio/speech",
        "v1/images/generations", "v1/images/edits", "v1/fine_tuning/jobs",
        "v1/moderations", "v1/files", "v1/assistants", "v1/threads",
        "v1/threads/runs", "v1/vector_stores", "v1/batches", "tokenize",
        "detokenize", "version", "health", "readyz", "livez", "models",
        "v1/responses"
    ]
    for ep in openai_compatible:
        keywords.add(ep)
        keywords.add(f"api/{ep}")

    # 3. AI Gateways & Multi-Model Proxies (LiteLLM, Portkey, Cloudflare AI Gateway)
    gateways = [
        "health/liveness", "health/readiness", "user/new", "user/info",
        "key/generate", "key/info", "key/delete", "model/info", "model/new",
        "spend/calculate", "global/spend", "team/new", "team/info",
        "customer/new", "v1/feedback", "v1/configs", "v1/prompts",
        "ai/v1/chat/completions", "gateway/v1/chat/completions"
    ]
    for ep in gateways:
        keywords.add(ep)

    # 4. Agentic & RAG Frameworks (LangServe, LangChain, Flowise, Dify, AnythingLLM)
    orchestrations = [
        "invoke", "batch", "stream", "stream_log", "input_schema",
        "output_schema", "config_schema", "playground", "c/docs",
        "c/invoke", "c/stream", "api/v1/prediction", "api/v1/chatflows",
        "api/v1/chatmessage", "api/v1/components-credentials", "api/v1/apikey",
        "api/v1/marketplaces", "api/v1/node-custom-functions", "v1/chat-messages",
        "v1/completion-messages", "v1/workflows/run", "v1/parameters",
        "v1/files/upload", "v1/audio-to-text", "v1/text-to-audio",
        "api/v1/workspace", "api/v1/workspaces", "api/v1/document",
        "api/v1/documents", "api/v1/chat", "api/agent/run", "api/agent/status",
        "api/agent/cancel", "api/tools/execute", "api/sessions"
    ]
    for ep in orchestrations:
        keywords.add(ep)

    # 5. Local WebUIs (Open WebUI, LibreChat, Text-Generation-WebUI)
    webuis = [
        "api/v1/auths/signin", "api/v1/auths/signup", "api/v1/users",
        "api/v1/chats", "api/v1/models", "api/v1/prompts", "api/v1/documents",
        "api/v1/tools", "api/v1/functions", "api/config", "api/models",
        "api/convos", "api/presets", "api/prompts", "api/endpoints",
        "api/v1/generate", "api/v1/chat", "api/v1/model", "api/v1/token-count"
    ]
    for ep in webuis:
        keywords.add(ep)

    # 6. Vector Databases & RAG Stores (Qdrant, ChromaDB, Milvus, Weaviate)
    vector_dbs = [
        "collections", "cluster", "cluster/metadata", "telemetry", "snapshots",
        "api/v1/collections", "api/v1/heartbeat", "api/v1/reset", "api/v1/databases",
        "v1/vector/search", "v1/vector/insert", "v1/vector/delete", "v1/vector/query",
        "v1/collection/create", "v1/meta", "v1/schema", "v1/nodes", "v1/objects",
        "v1/classifications", "v1/backups", "v1/.well-known/ready", "v1/.well-known/live",
        "indexes", "vectors/query", "vectors/fetch", "vectors/upsert"
    ]
    for ep in vector_dbs:
        keywords.add(ep)
        keywords.add(f"api/{ep}")

    # 7. AI Observability & Tracing (Langfuse, Arize Phoenix, Helicone)
    observability = [
        "api/public/events", "api/public/ingestion", "api/public/traces",
        "api/public/scores", "v1/traces", "v1/spans", "v1/evaluations",
        "helicone/v1/chat/completions"
    ]
    for ep in observability:
        keywords.add(ep)

    return sorted(list(keywords))


def generate_params_fuzzing():
    keywords = set()

    core_params = [
        "id", "user_id", "userid", "uid", "uuid", "guid", "account_id", "accountId",
        "org_id", "orgId", "tenant_id", "tenantId", "team_id", "project_id", "workspace_id",
        "client_id", "customer_id", "member_id", "order_id", "invoice_id", "payment_id",
        "transaction_id", "session_id", "token_id", "doc_id", "file_id", "item_id",
        "product_id", "post_id", "comment_id", "message_id", "thread_id", "ticket_id",
        "username", "user", "email", "mail", "phone", "mobile", "name", "fullname",
        "first_name", "last_name", "display_name", "handle", "slug", "alias",
        "password", "passwd", "pass", "pwd", "secret", "pin", "otp", "totp", "code",
        "token", "access_token", "refresh_token", "id_token", "auth_token", "api_key",
        "apikey", "api_secret", "client_secret", "jwt", "bearer", "session", "sid",
        "csrf", "csrf_token", "xsrf", "_token", "nonce", "state", "signature", "sig",
        "hash", "hmac", "digest", "checksum", "key", "public_key", "private_key",
        "redirect", "redirect_uri", "redirect_url", "return", "return_url", "return_to",
        "next", "next_url", "goto", "target", "dest", "destination", "continue",
        "callback", "callback_url", "cb", "jsonp", "forward", "ref", "referrer", "origin",
        "url", "uri", "link", "href", "src", "source", "endpoint", "webhook", "webhook_url",
        "host", "hostname", "domain", "subdomain", "port", "ip", "addr", "address",
        "file", "filename", "filepath", "path", "dir", "folder", "root", "base",
        "doc", "document", "page", "template", "tpl", "view", "layout", "theme",
        "include", "require", "import", "export", "load", "read", "fetch", "download",
        "upload", "attachment", "media", "image", "img", "avatar", "icon", "photo",
        "q", "query", "search", "s", "k", "keyword", "keywords", "term", "filter",
        "filters", "where", "sort", "sort_by", "order", "order_by", "asc", "desc",
        "group", "group_by", "limit", "offset", "cursor", "after", "before", "per_page",
        "page_size", "size", "count", "max", "min", "from", "to", "start", "end",
        "date", "start_date", "end_date", "since", "until", "timestamp", "ts", "ttl",
        "fields", "select", "columns", "expand", "embed", "populate", "relations",
        "format", "fmt", "type", "content_type", "mime", "encoding", "charset", "lang",
        "locale", "region", "country", "tz", "timezone", "currency", "unit",
        "action", "act", "cmd", "command", "exec", "run", "op", "operation", "method",
        "func", "function", "fn", "handler", "mode", "step", "stage", "phase",
        "status", "active", "enabled", "disabled", "verified", "approved", "deleted",
        "archived", "hidden", "public", "private", "internal", "visibility", "scope",
        "role", "roles", "permission", "permissions", "perm", "perms", "level", "tier",
        "plan", "is_admin", "isAdmin", "admin", "superuser", "staff", "moderator",
        "debug", "dev", "test", "testing", "staging", "sandbox", "dry_run", "dryRun",
        "verbose", "trace", "log", "logging", "profile", "bench", "inspect", "dump",
        "raw", "pretty", "indent", "nocache", "no_cache", "cache", "bust", "v", "ver",
        "version", "api_version", "revision", "rev", "build", "env", "environment",
        "config", "conf", "setting", "settings", "option", "options", "pref", "prefs",
        "feature", "flag", "flags", "experiment", "variant", "beta", "alpha", "preview",
        "draft", "override", "bypass", "force", "skip", "ignore", "allow", "deny",
        "su", "impersonate", "act_as", "on_behalf_of", "proxy", "via", "x_forwarded_for",
        "data", "payload", "body", "content", "text", "msg", "message", "title",
        "subject", "description", "summary", "notes", "comment", "reason", "tag", "tags",
        "label", "labels", "category", "cat", "categories", "meta", "metadata", "attrs",
        "attributes", "extra", "custom", "json", "xml", "yaml", "csv", "html", "sql",
        "eval", "expr", "expression", "script", "code_snippet", "formula", "regex",
        "pattern", "match", "replace", "split", "join", "prefix", "suffix", "delimiter",
        "port_num", "timeout", "retries", "delay", "interval", "rate", "concurrency",
        "async", "sync", "wait", "poll", "job_id", "task_id", "queue", "priority",
        "price", "amount", "total", "subtotal", "discount", "coupon", "promo", "tax",
        "fee", "balance", "credit", "debit", "qty", "quantity", "stock", "sku",
        "card", "cvv", "exp", "iban", "swift", "routing", "bank", "wallet", "address_id",
        "lat", "lon", "lng", "latitude", "longitude", "geo", "zip", "postal", "city",
        "model", "prompt", "system_prompt", "temperature", "max_tokens", "top_p", "stream",
        "assistant_id", "vector_store_id", "collection", "namespace", "index", "top_k",
        "schema", "table", "col", "row", "db", "database", "dsn", "connection",
        "recipient", "to_email", "cc", "bcc", "reply_to", "sender", "from_email",
        "oauth_token", "code_verifier", "code_challenge", "grant_type", "response_type",
        "redirect_state", "aud", "iss", "sub", "jti", "kid", "x5u", "jku"
    ]
    for p in core_params:
        keywords.add(p)

    prefixes = ["user", "admin", "api", "auth", "app", "sys", "dev", "test", "debug", "file", "url", "data", "req", "res", "meta", "cfg", "sec", "net", "db", "ai"]
    suffixes = ["id", "url", "uri", "path", "file", "key", "token", "secret", "mode", "flag", "type", "name", "host", "port", "code", "hash", "data", "json", "list", "info", "level", "state", "check", "bypass", "override", "enable", "disable"]
    for pref in prefixes:
        for suf in suffixes:
            keywords.add(f"{pref}_{suf}")

    # Common framework & WAF/debug query flags
    special = [
        "_method", "_rsc", "__proto__", "constructor", "prototype", "$where", "$regex",
        "$ne", "$gt", "XDEBUG_SESSION_START", "XDEBUG_PROFILE", "format=json",
        "alt=json", "prettyPrint", "fields=*", "include=*", "expand=*", "depth"
    ]
    for s in special:
        keywords.add(s)

    return sorted(list(keywords))


def generate_extensions_backups():
    keywords = set()

    web_exts = [
        ".php", ".php5", ".php7", ".php8", ".phtml", ".phar",
        ".asp", ".aspx", ".ashx", ".asmx", ".axd", ".config",
        ".jsp", ".jspx", ".do", ".action", ".jsf",
        ".html", ".htm", ".xhtml", ".shtml", ".cgi", ".pl", ".py", ".rb",
        ".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".vue", ".svelte",
        ".css", ".scss", ".less", ".map", ".js.map", ".css.map",
        ".json", ".json5", ".jsonl", ".ndjson", ".xml", ".xsd", ".xsl", ".xslt",
        ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg", ".cnf", ".properties",
        ".env", ".env.local", ".env.prod", ".env.dev", ".env.bak",
        ".graphql", ".gql", ".proto", ".wsdl", ".wadl", ".openapi", ".swagger"
    ]
    backup_exts = [
        ".bak", ".bak1", ".bak2", ".backup", ".bkp", ".old", ".orig", ".original",
        ".save", ".sav", ".tmp", ".temp", ".swp", ".swo", ".swn", "~",
        ".copy", ".dist", ".sample", ".example", ".default", ".template",
        ".disabled", ".off", ".unused", ".archive", ".prev", ".new", ".test", ".dev",
        ".stage", ".prod", ".local", ".inc", ".include", ".src", ".source",
        ".log", ".logs", ".out", ".err", ".trace", ".dump", ".dmp", ".core",
        ".sql", ".sql.gz", ".sql.bz2", ".sql.zip", ".sqlite", ".sqlite3", ".db", ".mdb",
        ".zip", ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".gz", ".bz2", ".xz", ".7z", ".rar",
        ".txt", ".md", ".rst", ".csv", ".tsv", ".pdf", ".doc", ".docx", ".xls", ".xlsx",
        ".key", ".pem", ".crt", ".cer", ".csr", ".der", ".p12", ".pfx", ".jks", ".keystore",
        ".lock", ".pid", ".sock", ".pid.lock"
    ]
    combo_bases = [".php", ".html", ".js", ".json", ".xml", ".yaml", ".yml", ".conf", ".config", ".env", ".sql", ".txt", ".aspx", ".jsp"]
    combo_sufs = [".bak", ".old", ".orig", ".save", ".swp", ".tmp", ".txt", ".gz", ".zip", "~", ".1", ".2"]

    for e in web_exts + backup_exts:
        keywords.add(e)
    for b in combo_bases:
        for s in combo_sufs:
            keywords.add(f"{b}{s}")

    return sorted(list(keywords))

PACKS = [
    {
        "id": "admin_portals",
        "category": "admin",
        "name": "Admin & Management Consoles (2025-2026)",
        "description": "Enterprise portals, obfuscated admin paths, hash prefixes, and control panels",
        "color": "#f43f5e",
        "generator": generate_admin_portals
    },
    {
        "id": "api_modern",
        "category": "api",
        "name": "Modern APIs, GraphQL & Actuator",
        "description": "REST v1-v4, OpenAPI/Swagger, GraphQL, tRPC, and Spring Boot 3 Actuator",
        "color": "#06b6d4",
        "generator": generate_api_modern
    },
    {
        "id": "ai_llm",
        "category": "ai",
        "name": "AI, LLMs & Agentic Ecosystem",
        "description": "Ollama, vLLM, LangServe, Flowise, LiteLLM, Vector DBs (Qdrant, Chroma), and AI Gateway endpoints",
        "color": "#a855f7",
        "generator": generate_ai_llm
    },
    {
        "id": "frameworks_backend",
        "category": "frameworks",
        "name": "Full-Stack Frameworks & Runtimes",
        "description": "Next.js App Router/RSC, Nuxt 3, SvelteKit, Laravel 11 Pulse, Django 5, Rails, and Go pprof",
        "color": "#3b82f6",
        "generator": generate_frameworks_backend
    },
    {
        "id": "cms_platforms",
        "category": "cms",
        "name": "CMS & Headless Platforms",
        "description": "WordPress, Strapi, Ghost, Directus, Kirby, and E-Commerce architectures",
        "color": "#8b5cf6",
        "generator": generate_cms_platforms
    },
    {
        "id": "cloud_devops",
        "category": "cloud",
        "name": "Cloud, K8s, Docker & Observability",
        "description": "Kubernetes API, Docker registry, Prometheus, Grafana, Traefik, Envoy, and Cloud Metadata",
        "color": "#10b981",
        "generator": generate_cloud_devops
    },
    {
        "id": "auth_sso",
        "category": "auth",
        "name": "Authentication, SSO & Tokens",
        "description": "OAuth 2.0, OIDC, SAML 2.0, Keycloak realms, magic links, and passkey endpoints",
        "color": "#ec4899",
        "generator": generate_auth_sso
    },
    {
        "id": "leaks_artifacts",
        "category": "leaks",
        "name": "Exposed Secrets, Git & Backups",
        "description": "Exposed .git, .env secrets, SQL database dumps, compressed archives, and debug logs",
        "color": "#f59e0b",
        "generator": generate_leaks_artifacts
    },
    {
        "id": "bypasses_heuristics",
        "category": "bypasses",
        "name": "403 Bypasses & Normalization Tricks",
        "description": "Path traversal tricks, semicolon normalization, trailing encodings, and debug parameter probes",
        "color": "#6366f1",
        "generator": generate_bypasses_heuristics
    },
    {
        "id": "params_fuzzing",
        "category": "params",
        "name": "HTTP & API Parameters (2026 Bug Bounty)",
        "description": "High-signal query, JSON body, IDOR, SSRF, debug, auth, and LLM parameter names for ?FUZZ=1 fuzzing",
        "color": "#14b8a6",
        "generator": generate_params_fuzzing
    },
    {
        "id": "extensions_backups",
        "category": "extensions",
        "name": "Extensions, Backups & Source Maps",
        "description": "Web runtimes, source maps (.js.map), config/key files, SQL dumps, and editor swap/backup suffixes",
        "color": "#f97316",
        "generator": generate_extensions_backups
    }
]

def main():
    total_keywords = 0
    total_raw_bytes = 0
    total_gz_bytes = 0

    print("=== Generating Curated Wordlist Packs ===")
    for pack in PACKS:
        keywords = pack["generator"]()
        count = len(keywords)
        total_keywords += count

        pack_data = {
            "id": pack["id"],
            "category": pack["category"],
            "name": pack["name"],
            "description": pack["description"],
            "version": "1.0.0",
            "color": pack["color"],
            "count": count,
            "keywords": keywords
        }

        json_bytes = json.dumps(pack_data, indent=2, ensure_ascii=False).encode("utf-8")
        total_raw_bytes += len(json_bytes)

        # Write standalone JSON
        json_path = os.path.join(JSON_OUT_DIR, f"{pack['id']}.json")
        with open(json_path, "wb") as f:
            f.write(json_bytes)

        # Write compressed .json.gz (level 9 maximum compression)
        gz_path = os.path.join(GZ_OUT_DIR, f"{pack['id']}.json.gz")
        with gzip.GzipFile(gz_path, "wb", compresslevel=9, mtime=0) as f:
            f.write(json_bytes)

        gz_size = os.path.getsize(gz_path)
        total_gz_bytes += gz_size
        ratio = (1 - gz_size / len(json_bytes)) * 100

        print(f"[*] {pack['name']}: {count:,} keywords | Raw: {len(json_bytes)/1024:.1f} KB | Gz: {gz_size/1024:.1f} KB ({ratio:.1f}% saved)")

    print("--------------------------------------------------")
    print(f"Total Keywords: {total_keywords:,}")
    print(f"Total Raw JSON Size: {total_raw_bytes/1024:.1f} KB ({total_raw_bytes/(1024*1024):.2f} MB)")
    print(f"Total Embedded Gzip Size: {total_gz_bytes/1024:.1f} KB ({total_gz_bytes/(1024*1024):.2f} MB)")
    print(f"Overall Compression Ratio: {(1 - total_gz_bytes / total_raw_bytes) * 100:.1f}%")
    print("=== Generation Complete! ===")

if __name__ == "__main__":
    main()
