services:
  app:
    image: "{{services_image}}"
    container_name: swiftdeploy-app
    environment:
      - MODE={{services_mode}}
      - APP_VERSION={{services_version}}
      - APP_PORT={{services_port}}
    volumes:
      - app-logs:/app/logs
    networks:
      - "{{network_name}}"
    restart: "{{restart_policy}}"
    healthcheck:
      test:
        ["CMD", "wget", "-qO-", "http://localhost:{{services_port}}/healthz"]
      interval: 10s
      timeout: 3s
      start_period: 5s
      retries: 3
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true

  nginx:
    image: "{{nginx_image}}"
    container_name: swiftdeploy-nginx
    ports:
      - "{{nginx_port}}:{{nginx_port}}"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - nginx-logs:/var/log/nginx
    networks:
      - "{{network_name}}"
    restart: unless-stopped
    depends_on:
      app:
        condition: service_healthy
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
      - CHOWN
      - SETUID
      - SETGID
      - DAC_OVERRIDE

volumes:
  app-logs:
  nginx-logs:

networks:
  "{{network_name}}":
    driver: "{{network_driver}}"