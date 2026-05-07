log_format custom '$time_iso8601 | $status | ${request_time}s | $upstream_addr | $request';

upstream app_backend {
    server app:{{services_port}};
}

server {
    listen {{nginx_port}};
    server_name _;

    access_log /var/log/nginx/access.log custom;

    proxy_connect_timeout {{proxy_timeout}}s;
    proxy_send_timeout {{proxy_timeout}}s;
    proxy_read_timeout {{proxy_timeout}}s;

    proxy_pass_header X-Mode;
    add_header X-Deployed-By swiftdeploy always;

    location / {
        proxy_pass http://app_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    error_page 502 /error502.json;
    error_page 503 /error503.json;
    error_page 504 /error504.json;

    location = /error502.json {
        internal;
        default_type application/json;
        return 502 '{"error":"Bad Gateway","code":"502","service":"swiftdeploy","contact":"admin@swiftdeploy.local"}';
    }

    location = /error503.json {
        internal;
        default_type application/json;
        return 503 '{"error":"Service Unavailable","code":"503","service":"swiftdeploy","contact":"admin@swiftdeploy.local"}';
    }

    location = /error504.json {
        internal;
        default_type application/json;
        return 504 '{"error":"Gateway Timeout","code":"504","service":"swiftdeploy","contact":"admin@swiftdeploy.local"}';
    }
}
