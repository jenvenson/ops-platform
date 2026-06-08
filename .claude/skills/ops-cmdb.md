# CMDB Management Skill (`ops-cmdb`)

## Description
Manage configuration items in the Configuration Management Database including projects, environments, and servers.

## Functions
- Create/update/delete projects
- Manage environment configurations
- Register and update server information
- Link servers to projects/environments
- Query CMDB for infrastructure information

## Usage
When you need to manage infrastructure configuration or query CMDB information, use this skill to interact with the configuration management database.

### Examples:
- "Create a new project called 'my-project'"
- "Add a server with IP 192.168.1.100 to environment 'dev'"
- "List all servers in the 'prod' environment"
- "Link server 'web01' to project 'ecommerce'"

## Commands
1. **Project Management:**
   - `create_project(name, description, env_id)`
   - `update_project(project_id, name, description)`
   - `delete_project(project_id)`
   - `list_projects()`

2. **Environment Management:**
   - `create_environment(name, description)`
   - `update_environment(env_id, name, description)`
   - `list_environments()`

3. **Server Management:**
   - `register_server(ip, hostname, os, env_ids, project_id)`
   - `update_server(server_id, params)`
   - `get_server_details(server_id)`
   - `list_servers(environment="", project="")`

4. **Queries:**
   - `find_servers_by_ip(ip_prefix)`
   - `find_servers_by_env(environment)`
   - `find_servers_by_project(project)`
   - `get_server_inventory()`

## Integration Notes
This skill interacts with the backend CMDB module at `/api/cmdb/` endpoints and uses the data models defined in `/backend/internal/cmdb/models.go`.