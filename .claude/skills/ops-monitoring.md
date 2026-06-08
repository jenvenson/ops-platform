# Monitoring Skill (`ops-monitoring`)

## Description
Handle system monitoring and metrics collection.

## Functions
- View system metrics
- Access Grafana dashboards
- Monitor host performance
- Check system health
- View performance warnings
- Access monitoring big screen

## Usage
When you need to check system health, view metrics, or monitor infrastructure performance, use this skill to interact with the monitoring system.

### Examples:
- "Show me the current CPU usage across all hosts"
- "Display the monitoring dashboard for 'prod' environment"
- "Check the health status of all monitored hosts"
- "Show performance warnings for disk usage"

## Commands
1. **System Metrics:**
   - `get_system_metrics(host_id="", metric_type="cpu")`
   - `get_cpu_usage(host_id="")`
   - `get_memory_usage(host_id="")`
   - `get_disk_usage(host_id="")`
   - `get_network_stats(host_id="")`

2. **Host Monitoring:**
   - `get_host_list()`
   - `get_host_details(host_id)`
   - `check_host_health(host_id)`
   - `get_hosts_by_status(status="online")`

3. **Performance Warnings:**
   - `get_performance_warnings(threshold=80)`
   - `get_cpu_warnings(threshold=80)`
   - `get_memory_warnings(threshold=80)`
   - `get_disk_warnings(threshold=80)`

4. **Grafana Integration:**
   - `get_grafana_dashboards()`
   - `open_dashboard(dashboard_id)`
   - `get_dashboard_embed_url(dashboard_id)`

5. **Health Checks:**
   - `get_system_health()`
   - `get_monitoring_agents_status()`
   - `run_custom_query(query)`

## Integration Notes
This skill interacts with the monitoring module at `/api/monitor/` endpoints and integrates with Prometheus for metrics collection. The system uses Grafana for visualization and provides both individual host metrics and aggregated system views.