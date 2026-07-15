FROM grafana/grafana:11.1.0
# Disable setup wizard / password change on first run
ENV GF_SECURITY_ADMIN_PASSWORD=admin
ENV GF_USERS_ALLOW_SIGN_UP=false
COPY provisioning/datasources/datasources.yml /etc/grafana/provisioning/datasources/datasources.yml
COPY provisioning/dashboards/dashboards.yml /etc/grafana/provisioning/dashboards/dashboards.yml
COPY dashboards/dashboard.json /etc/grafana/dashboards/dashboard.json
EXPOSE 3000
